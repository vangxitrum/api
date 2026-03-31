package storage

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

type StorageHelper interface {
	Upload(ctx context.Context, data string, reader io.Reader) (*Object, error)
	PackUploadByte(ctx context.Context, name string, data []byte) (*Object, error)
	Uploads(
		ctx context.Context,
		data string,
		fileInfos map[string]io.Reader,
	) ([]*Object, int64, error)
	UploadZip(ctx context.Context, data string, reader io.Reader) (*Object, error)
	UploadRaw(ctx context.Context, data string, size int64, reader io.Reader) (*Object, error)

	Delete(ctx context.Context, object *Object) error
	Download(ctx context.Context, object *Object) (io.Reader, error)

	GetLink(ctx context.Context, object *Object) (string, int64, error)
	GetAIOZPrice(ctx context.Context) (float64, error)
	GetTranscodeStatus(ctx context.Context, fileId string) (string, error)
	GetZipHeader(ctx context.Context, fileId string) (*zipHeader, error)
	GetFileRecord(ctx context.Context, fileId string) (*GetFileRecordResponse, error)

	Transcode(context.Context, string, *TranscodeProfile) (*TranscodeResponse, error)
	GetDetailBalance(context.Context) (*GetDetailBalanceResponse, error)
}

type Object struct {
	Id     string
	Offset int64
	Size   int64
	Name   string
}

type CdnFile struct {
	BelongsToId uuid.UUID `json:"source_id"`
	FileId      string    `json:"file_id"`
	Size        int64     `json:"size"`
	Type        string    `json:"type"`
	Offset      int64     `json:"offset"`
	CreatedAt   time.Time `json:"created_at"`
}
