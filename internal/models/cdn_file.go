package models

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

var (
	CdnZipFileType       = "zip"
	CdnThumbnailType     = "thumbnail"
	CdnSourceFileType    = "source"
	CdnM3u8FileType      = "m3u8"
	CdnWatermarkType     = "watermark"
	CdnLogoType          = "logo"
	CdnVideoContentType  = "video_content"
	CdnAudioContentType  = "audio_content"
	CdnVideoPlaylistType = "video_playlist"
	CdnAudioPlaylistType = "audio_playlist"
	CdnMp4Type           = "mp4"
	CdnCaptionType       = "caption"
	CdnCaptionM3u8Type   = "caption_m3u8"
	CdnChapterType       = "chapter"
	CdnAudioType         = "audio"
)

type CdnFileRepository interface {
	Create(context.Context, *CdnFile) error

	GetCdnFiles(context.Context) ([]*CdnFile, error)
	GetCdnFileByFileId(context.Context, uuid.UUID) (*CdnFile, error)
	GetUserTotalStorage(context.Context, uuid.UUID) (int64, error)
	GetMediaTotalStorage(context.Context, uuid.UUID) (int64, error)
	GetCdnFilesByType(context.Context, string) ([]*CdnFile, error)
	GetTotalSizeStorage(context.Context) (int64, error)

	UpdateCdnFile(context.Context, *CdnFile) error

	DeleteCdnFileByFileId(
		context.Context, string,
	) error
}

type CdnFile struct {
	Id        string    `json:"file_id"    gorm:"primaryKey"`
	Size      int64     `json:"size"`
	Type      string    `json:"type"`
	Offset    int64     `json:"offset"`
	Index     int       `json:"index"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy uuid.UUID `json:"created_by"`
}

func NewCdnFile(
	userId uuid.UUID,
	fileId string,
	size, offset int64,
	index int,
	cdnFileType string,
) *CdnFile {
	return &CdnFile{
		Id:        fileId,
		Size:      size,
		Offset:    offset,
		Index:     index,
		Type:      cdnFileType,
		CreatedAt: time.Now().UTC(),
		CreatedBy: userId,
	}
}

type FileInfo struct {
	RedirectUrl string
	MimeType    string
	ExpiredAt   int64
	Reader      io.Reader
	Size        int64
	MediaId     uuid.UUID
	UserId      uuid.UUID
}

func NewFileInfo(
	redirectUrl, contentType string,
	reader io.Reader,
	size, expiredAt int64,
	mediaId, userId uuid.UUID,
) *FileInfo {
	return &FileInfo{
		RedirectUrl: redirectUrl,
		MimeType:    contentType,
		Reader:      reader,
		Size:        size,
		ExpiredAt:   expiredAt,
		MediaId:     mediaId,
		UserId:      userId,
	}
}

type FileInfoBuilder struct {
	info FileInfo
}

func (b *FileInfoBuilder) SetRedirectUrl(redirectUrl string) *FileInfoBuilder {
	b.info.RedirectUrl = redirectUrl
	return b
}

func (b *FileInfoBuilder) SetMimeType(mimeType string) *FileInfoBuilder {
	b.info.MimeType = mimeType
	return b
}

func (b *FileInfoBuilder) SetReader(reader io.Reader) *FileInfoBuilder {
	b.info.Reader = reader
	return b
}

func (b *FileInfoBuilder) SetSize(size int64) *FileInfoBuilder {
	b.info.Size = size
	return b
}

func (b *FileInfoBuilder) SetExpiredAt(expiredAt int64) *FileInfoBuilder {
	b.info.ExpiredAt = expiredAt
	return b
}

func (b *FileInfoBuilder) SetMediaId(mediaId uuid.UUID) *FileInfoBuilder {
	b.info.MediaId = mediaId
	return b
}

func (b *FileInfoBuilder) SetUserId(userId uuid.UUID) *FileInfoBuilder {
	b.info.UserId = userId
	return b
}

func (b *FileInfoBuilder) Build(ctx context.Context) *FileInfo {
	if b.info.UserId == uuid.Nil || b.info.MediaId == uuid.Nil {
		slog.WarnContext(ctx, "invalid file info data")
	}

	if b.info.RedirectUrl != "" && b.info.ExpiredAt == 0 {
		b.info.ExpiredAt = time.Now().Add(1 * time.Minute).UnixNano()
		slog.WarnContext(ctx, "expired at is not set, using default value")
	}

	return &b.info
}
