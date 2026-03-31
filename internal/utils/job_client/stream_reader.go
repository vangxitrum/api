package client

import (
	"crypto/md5"
	"fmt"
	"io"

	job_pb "10.0.0.50/tuan.quang.tran/vms-v2/internal/proto/job"
)

func NewStreamReader(stream job_pb.JobService_DownloadMediaResourceClient) *StreamReader {
	return &StreamReader{
		stream: stream,
		buffer: nil,
		offset: 0,
	}
}

func (r *StreamReader) Read(p []byte) (n int, err error) {
	// If we've consumed the current buffer, get the next chunk
	if r.buffer == nil || r.offset >= len(r.buffer) {
		chunk, err := r.stream.Recv()
		if err != nil {
			if err == io.EOF {
				return 0, io.EOF
			}
			return 0, fmt.Errorf("error receiving chunk: %w", err)
		}

		if len(chunk.Data) == 0 {
			return 0, fmt.Errorf("empty chunk received")
		}

		if chunk.BlockMetadata == nil {
			return 0, fmt.Errorf("block metadata is nil")
		}

		if chunk.BlockMetadata.Size != int32(len(chunk.Data)) {
			return 0, fmt.Errorf("chunk size mismatch")
		}

		if fmt.Sprintf("%x", md5.Sum(chunk.Data)) != chunk.BlockMetadata.Checksum {
			return 0, fmt.Errorf("checksum mismatch")
		}

		r.buffer = chunk.Data
		r.offset = 0
	}

	// Copy the data from our buffer to the target buffer
	n = copy(p, r.buffer[r.offset:])
	r.offset += n
	return n, nil
}
