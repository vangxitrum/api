package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/proto/grpc_service"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

var _ grpc_service.GRPCServiceServer = (*GRPCService)(nil)

type GRPCService struct {
	grpc_service.UnimplementedGRPCServiceServer
	mediaRepo        models.MediaRepository
	mediaCaptionRepo models.MediaCaptionRepository
	cdnRepo          models.CdnFileRepository
	storageHelper    storage.StorageHelper

	mediaService *services.MediaService

	storagePath string
	outputPath  string
}

func NewGRPCService(
	mediaRepo models.MediaRepository,
	mediaCaptionRepo models.MediaCaptionRepository,
	cdnRepo models.CdnFileRepository,
	storageHelper storage.StorageHelper,
	storagePath string,
	outputPath string,
	mediaService *services.MediaService,
) *GRPCService {
	return &GRPCService{
		mediaRepo:        mediaRepo,
		mediaCaptionRepo: mediaCaptionRepo,
		cdnRepo:          cdnRepo,
		storageHelper:    storageHelper,
		storagePath:      storagePath,
		outputPath:       outputPath,
		mediaService:     mediaService,
	}
}

type MediaInfoRequest struct {
	Id     string `json:"id"`
	Lang   string `json:"lang"`
	Action string `json:"action"`
}

func (j *GRPCService) Ping(
	ctx context.Context,
	req *grpc_service.PingRequest,
) (*grpc_service.PingResponse, error) {
	return &grpc_service.PingResponse{
		Message: "pong",
	}, nil
}

func (j *GRPCService) UploadMedia(stream grpc_service.GRPCService_UploadMediaServer) error {
	var (
		metadata *grpc_service.BlockMetadata
		fileInfo *grpc_service.FileInfo
		mediaId  uuid.UUID = uuid.Nil
		saveFunc services.SaveFunc
	)

	var totalSize int64 = 0
	for {
		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				if saveFunc != nil {
					if err := saveFunc(stream.Context(), nil, true); err != nil {
						return status.Errorf(codes.Internal, "Failed to save file: %v", err)
					}
				}

				break
			}

			return status.Errorf(codes.Internal, "Failed to receive file: %v", err)
		}

		if fileInfo == nil {
			fileInfo = req.GetFileInfo()
			if fileInfo == nil {
				return status.Errorf(codes.InvalidArgument, "File info is required.")
			}

			if fileInfo.Type == "" {
				return status.Errorf(codes.InvalidArgument, "File type is required.")
			}

			if fileInfo.Size == 0 {
				return status.Errorf(codes.InvalidArgument, "File size is required.")
			}

			if fileInfo.Size > models.MaxMediaSize {
				return status.Errorf(codes.InvalidArgument, "File size exceed max file size.")
			}
		}

		metadata = req.GetBlockMetadata()
		if metadata == nil {
			return status.Errorf(codes.InvalidArgument, "Block metadata is required.")
		}

		if mediaId == uuid.Nil {
			mediaId, err = uuid.Parse(req.GetMediaId())
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "Invalid media id.")
			}

		}

		if saveFunc == nil {
			saveFunc, err = j.mediaService.GetSaveJobResourceFunc(
				stream.Context(),
				mediaId,
			)
			if err != nil {
				return err
			}
		}

		block := req.GetData()
		if len(block) == 0 {
			return status.Errorf(codes.InvalidArgument, "Block empty.")
		}

		if len(block) != int(metadata.Size) {
			return status.Errorf(codes.InvalidArgument, "Block size mismatch.")
		}

		hasher := md5.New()
		if _, err := io.Copy(
			hasher,
			bytes.NewReader(block),
		); err != nil {
			return status.Errorf(codes.Internal, "Failed to hash block: %v", err)
		}

		if metadata.Checksum != fmt.Sprintf("%x", hasher.Sum(nil)) {
			return status.Errorf(codes.InvalidArgument, "Checksum fail.")
		}

		if err := saveFunc(stream.Context(), block, false); err != nil {
			return status.Errorf(codes.Internal, "Failed to save block: %v", err)
		}

		totalSize += int64(metadata.Size)
		if totalSize > fileInfo.Size {
			return status.Errorf(codes.InvalidArgument, "File size exceed declared size.")
		}
	}

	if metadata == nil {
		return status.Errorf(codes.InvalidArgument, "Metadata is required.")
	}

	if mediaId == uuid.Nil {
		return status.Errorf(codes.InvalidArgument, "Job id is required.")
	}

	return stream.SendAndClose(&grpc_service.UploadMediaResponse{
		MediaId: mediaId.String(),
	})
}

func (s *GRPCService) DownloadAudioFile(
	req *grpc_service.MediaInfoRequest,
	stream grpc_service.GRPCService_DownloadAudioFileServer,
) error {
	slog.Info("received from grpc:", slog.Any("media_id", req.Id), slog.Any("lang", req.Lang))
	mediaFolder := fmt.Sprintf("%s/%s", s.storagePath, req.Id)
	files, err := os.ReadDir(mediaFolder)
	if err != nil {
		slog.Error("reading media folder", slog.Any("error", err))
		err := stream.Send(&grpc_service.AudioFileChunk{
			Data:        nil,
			ChunkNumber: -1,
			TotalChunks: 0,
		})
		if err != nil {
			slog.Error("send error notification to transcribe service", slog.Any("error", err))
			return fmt.Errorf("notify transcribe service: %w", err)
		}
		return err
	}

	var audioFilePath string
	for _, file := range files {
		if filepath.Ext(file.Name()) == fmt.Sprintf(".%s", models.AudioFormat) {
			audioFilePath = filepath.Join(mediaFolder, file.Name())
			break
		}
	}

	if audioFilePath == "" {
		slog.Error("no audio file found in folder", slog.Any("error", err))
		return fmt.Errorf("no audio file found in folder %s", mediaFolder)
	}

	file, err := os.Open(audioFilePath)
	if err != nil {
		slog.Error("opening file", slog.Any("file", audioFilePath))
		return err
	}

	defer file.Close()
	chunkSize := int32(1 * 1024 * 1024)
	buf := make([]byte, chunkSize)
	chunkNumber := req.StartChunk
	fileInfo, err := file.Stat()
	if err != nil {
		slog.Error("get file info", slog.Any("error", err))
		return err
	}

	totalChunks := int32(fileInfo.Size() / int64(chunkSize))
	if fileInfo.Size()%int64(chunkSize) > 0 {
		totalChunks++
	}

	offset := int64((req.StartChunk - 1) * chunkSize)
	if offset < 0 || offset >= fileInfo.Size() {
		slog.Error(
			"invalid file offset",
			slog.Any("offset", offset),
			slog.Any("fileSize", fileInfo.Size()),
		)
		return fmt.Errorf(
			"invalid offset: %d (must be between 0 and %d)",
			offset,
			fileInfo.Size()-1,
		)
	}

	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		slog.Error("seek file", slog.Any("error", err))
		return err
	}

	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}

		if err != nil {
			slog.Error("reading file", slog.Any("error", err))
			return err
		}

		err = stream.Send(&grpc_service.AudioFileChunk{
			Data:        buf[:n],
			ChunkNumber: int32(chunkNumber),
			TotalChunks: totalChunks,
		})
		if err != nil {
			slog.Error("sending chunk", slog.Any("error", err))
			return err
		}

		chunkNumber++
	}

	return nil
}

func (s *GRPCService) UploadVTTFiles(
	ctx context.Context,
	req *grpc_service.UploadVTTFilesRequest,
) (*grpc_service.UploadVTTFilesResponse, error) {
	return &grpc_service.UploadVTTFilesResponse{
		Message: fmt.Sprintf("successfully receive %d file(s)", len(req.Files)),
	}, nil
}
