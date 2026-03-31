package client

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	job_pb "10.0.0.50/tuan.quang.tran/vms-v2/internal/proto/job"
)

var (
	jobClientConfig = fmt.Sprintf(`
        {
	        "loadBalancingPolicy": "round_robin",
	        "healthCheckConfig": {
		        "serviceName": ""
	        }
	    }`)
	kacp = keepalive.ClientParameters{
		Time:                10 * time.Second, // send pings every 10 seconds if there is no activity
		Timeout:             1 * time.Second,  // wait 1 second for ping ack before considering the connection dead
		PermitWithoutStream: true,             // send pings even without active streams
	}
)

type JobClient struct {
	addr string

	client job_pb.JobServiceClient
	conn   *grpc.ClientConn
}

func NewJobClient(addr string) (*JobClient, error) {
	cb := GrpcConnBuilder{}
	cb.WithInsecure()
	cb.WithContext(context.Background())
	cb.WithOptions(
		grpc.WithDefaultServiceConfig(jobClientConfig),
		grpc.WithKeepaliveParams(kacp),
	)

	conn, err := cb.GetConn(addr)
	if err != nil {
		return nil, err
	}

	return &JobClient{
		addr: addr,

		client: job_pb.NewJobServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *JobClient) RegisterJob(
	ctx context.Context,
	input *job_pb.RegisterJobRequest,
) (*job_pb.Job, error) {
	resp, err := c.client.RegisterJob(ctx, input)
	if err != nil {
		return nil, err
	}

	return resp.Job, nil
}

func (c *JobClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

func (c *JobClient) UploadMediaResource(
	ctx context.Context,
	jobId string,
	fileName string,
	size int64,
	reader io.Reader,
) error {
	stream, err := c.client.UploadMediaResource(ctx)
	if err != nil {
		return err
	}

	for {
		buf := make([]byte, 1024)
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		md5Hash := md5.Sum(buf[:n])
		if err := stream.Send(&job_pb.UploadMediaResourceRequest{
			JobId: jobId,
			Data:  buf[:n],
			BlockMetadata: &job_pb.BlockMetadata{
				Size:     int32(n),
				Checksum: fmt.Sprintf("%x", md5Hash),
			},
			FileInfo: &job_pb.FileInfo{
				Name: fileName,
				Size: size,
				Type: "source",
			},
		}); err != nil {
			return err
		}
	}

	if _, err := stream.CloseAndRecv(); err != nil {
		return err
	}

	return nil
}

func (c *JobClient) GetFileReader(
	ctx context.Context,
	jobId string,
	fileType string,
) (io.ReadCloser, error) {
	panic("not implemented")
}

func (c *JobClient) GetJobStatus(
	ctx context.Context,
	jobId string,
) (string, error) {
	resp, err := c.client.GetJobStatus(ctx, &job_pb.GetJobStatusRequest{
		JobId: jobId,
	})
	if err != nil {
		return "", err
	}

	return resp.Status, nil
}

func (c *JobClient) GetJobDetail(
	ctx context.Context,
	jobId, registerId string,
) (*job_pb.Job, error) {
	resp, err := c.client.GetJobDetail(ctx, &job_pb.GetJobDetailRequest{
		JobId:      jobId,
		RegisterId: registerId,
	})
	if err != nil {
		return nil, err
	}

	return resp.Job, nil
}

func (c *JobClient) GetJob(ctx context.Context, workerId string) (*job_pb.Job, error) {
	resp, err := c.client.GetJob(ctx, &job_pb.GetJobRequest{
		WorkerId: workerId,
	})
	if err != nil {
		return nil, err
	}

	return resp.Job, nil
}

type StreamReader struct {
	stream job_pb.JobService_DownloadMediaResourceClient
	buffer []byte
	offset int

	totalSize int64
}

func (c *JobClient) DownloadJobResource(
	ctx context.Context,
	jobId, fileType string,
) (io.Reader, error) {
	stream, err := c.client.DownloadMediaResource(ctx, &job_pb.DownloadMediaResourceRequest{
		JobId:    jobId,
		WorkerId: "worker-id",
		Type:     "source",
	})
	if err != nil {
		return nil, err
	}

	return NewStreamReader(stream), nil
}

func (c *JobClient) CompletePlaylist(
	ctx context.Context,
	workerId string,
	pl *job_pb.Playlist,
) error {
	if _, err := c.client.CompletePlaylist(ctx, &job_pb.CompletePlaylistRequest{
		WorkerId: workerId,
		Playlist: pl,
	}); err != nil {
		return err
	}

	return nil
}

func (c *JobClient) GetPlaylists(
	ctx context.Context,
	registerId, status string,
	cursor int64,
) ([]*job_pb.Playlist, int64, error) {
	resp, err := c.client.GetPlaylists(ctx, &job_pb.GetPlaylistsRequest{
		RegisterId: registerId,
		Cursor:     cursor,
		Status:     status,
	})
	if err != nil {
		return nil, 0, err
	}

	return resp.Playlists, resp.NextCursor, nil
}
