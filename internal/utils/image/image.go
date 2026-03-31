package image

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/mdobak/go-xerrors"
	"golang.org/x/image/draw"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
)

type ThumbnailHelper struct {
	storageHelper storage.StorageHelper
}

func NewThumbnailHelper(
	storageHelper storage.StorageHelper,
) *ThumbnailHelper {
	return &ThumbnailHelper{
		storageHelper: storageHelper,
	}
}

func (h *ThumbnailHelper) GenerateThumbnail(
	ctx context.Context,
	userId uuid.UUID,
	src io.ReadSeeker,
) (*models.Thumbnail, int64, error) {
	thumbnail := models.NewThumbnail()
	thumbnail.Resolutions = make([]*models.ThumbnailResolution, 0, len(models.DefaultThumbnailSize))
	resolutionMapping := make(map[string]io.Reader)
	for _, resolution := range models.DefaultThumbnailSize {
		if _, err := src.Seek(0, io.SeekStart); err != nil {
			return nil, 0, err
		}

		reader, err := processImage(src, resolution.Width, resolution.Height)
		if err != nil {
			return nil, 0, err
		}

		resolutionMapping[fmt.Sprintf("%dx%d", resolution.Width, resolution.Height)] = reader
	}

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, 0, err
	}

	resolutionMapping["original"] = src
	objs, total, err := h.storageHelper.Uploads(ctx, thumbnail.Id.String(), resolutionMapping)
	if err != nil {
		return nil, 0, err
	}

	var fileId string
	for _, obj := range objs {
		if fileId == "" {
			fileId = obj.Id
		}

		thumbnail.Resolutions = append(
			thumbnail.Resolutions,
			models.NewThumbnailResolution(
				thumbnail.Id,
				obj.Name,
				obj.Size,
				obj.Offset,
			),
		)
	}
	file := models.NewCdnFile(userId, fileId, total, 0, 1, models.CdnThumbnailType)
	thumbnail.File = &models.ThumbnailFile{
		FileId:      fileId,
		ThumbnailId: thumbnail.Id,
		File:        file,
	}

	return thumbnail, total, nil
}

func processImage(
	src io.Reader, maxWidth, maxHeight int,
) (io.Reader, error) {
	img, format, err := image.Decode(src)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()
	var newWidth, newHeight int
	ratio := float64(origWidth) / float64(origHeight)

	if origWidth > origHeight {
		if origWidth > maxWidth {
			newWidth = maxWidth
			newHeight = int(float64(maxWidth) / ratio)
		} else {
			newWidth, newHeight = origWidth, origHeight
		}
	} else {
		if origHeight > maxHeight {
			newHeight = maxHeight
			newWidth = int(float64(maxHeight) * ratio)
		} else {
			newWidth, newHeight = origWidth, origHeight
		}
	}
	newImg := image.NewRGBA(
		image.Rect(
			0,
			0,
			newWidth,
			newHeight,
		),
	)
	draw.CatmullRom.Scale(
		newImg,
		newImg.Bounds(),
		img,
		img.Bounds(),
		draw.Over,
		nil,
	)

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		switch strings.ToLower(format) {
		case "jpeg":
			err = jpeg.Encode(pw, newImg, nil)
		case "png":
			err = png.Encode(pw, newImg)
		default:
			err := pw.CloseWithError(xerrors.New("unsupported image format"))
			if err != nil {
				return
			}
			return
		}
		if err != nil {
			err := pw.CloseWithError(err)
			if err != nil {
				return
			}
		}
	}()

	return pr, nil
}
