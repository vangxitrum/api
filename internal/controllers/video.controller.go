package controllers

import (
	"crypto/md5"
	"fmt"
	"io"
	"log/slog"
	"math"
	"mime/multipart"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

type MediaController struct {
	mediaService *services.MediaService
	usageService *services.UsageService
}

func NewMediaController(
	mediaService *services.MediaService,
	usageService *services.UsageService,
) *MediaController {
	return &MediaController{
		mediaService: mediaService,
		usageService: usageService,
	}
}

type MediaWatermark struct {
	Id      string `json:"id"`
	Top     string `json:"top"`
	Left    string `json:"left"`
	Bottom  string `json:"bottom"`
	Right   string `json:"right"`
	Width   string `json:"width"`
	Height  string `json:"height"`
	Opacity string `json:"opacity"`
} //	@name	MediaWatermark

type CreateMediaRequest struct {
	// Title of the media
	Title string `json:"title"            form:"title"`
	// Description of the media
	Description string `json:"description"      form:"description"`
	// // Is panoramic media
	// IsPanoramic *bool `json:"is_panoramic" form:"is_panoramic"`
	// Is public media
	IsPublic *bool `json:"is_public"        form:"is_public"`
	// Metadata of the media (key-value pair, max: 50 items, key max length: 255, value max length: 255)
	Metadata []models.Metadata `json:"metadata"         form:"metadata"`
	// Qualities of the media (default: 1080p, 720p,  360p, allow:2160p, 1440p, 1080p, 720p,  360p, 240p, 144p)
	Qualities []*models.QualityConfig `json:"qualities"        form:"qualities"`
	// Type of the media (default: video, allowed: video, audio)
	Type string `json:"type"             form:"type"`
	// Tags of the media (max: 50 items, max length: 255)
	Tags []string `json:"tags"             form:"tags"`
	// Media thumbnailConfig
	Watermark *MediaWatermark `json:"watermark"        form:"watermark"`
	// SegmentConfig
	SegmentDuration int32 `json:"segment_duration" form:"segment_duration"`
} //	@name	CreateMediaRequest

type CreateMediaResponse struct {
	Status string              `json:"status"`
	Data   *models.MediaObject `json:"data"`
} //	@name	CreateMediaResponse

// CreateMediaObject   godoc
//
//	@Summary			Create media object
//	@Description		Create a media object
//	@Tags				media
//	@Id					POST-media
//	@Security			BasicAuth
//	@Security			Bearer
//	@Accept				json
//	@Accept				x-www-form-urlencoded
//	@Produce			json
//	@Param				request	body		CreateMediaRequest		true	"media's info"
//	@Success			200		{object}	CreateMediaResponse		"success"
//	@Header				200		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				200		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				200		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			400		{object}	models.ResponseError
//	@Header				400		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			403		{object}	models.ResponseError
//	@Header				403		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				403		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				403		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			404		{object}	models.ResponseError
//	@Header				404		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				404		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				404		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500		{object}	models.ResponseError
//	@Header				500		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router				/media/create [post]
//	@x-client-action	"create"
func (c *MediaController) CreateMediaObject(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CreateMediaObject").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	var payload CreateMediaRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				"Invalid input.",
			),
		)
	}

	payload.Title = strings.TrimSpace(payload.Title)
	if payload.Title == "" {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				"Title is required.",
			),
		)
	}

	if len(payload.Title) > models.TitleMaxLen {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				fmt.Sprintf(
					"Title length must be less than %d characters.",
					models.TitleMaxLen,
				),
			),
		)
	}

	payload.Description = strings.TrimSpace(payload.Description)
	if len(payload.Description) > models.DescriptionMaxLen {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				fmt.Sprintf(
					"Description length must be less than %d characters.",
					models.DescriptionMaxLen,
				),
			),
		)
	}

	payload.Type = strings.TrimSpace(payload.Type)
	if len(payload.Type) != 0 {
		if payload.Type != models.VideoType &&
			payload.Type != models.AudioType {
			return response.ResponseFailMessage(
				ctx,
				http.StatusBadRequest,
				"Invalid media type.",
			)
		}
	} else {
		payload.Type = models.VideoType
	}

	var isPublic bool
	// if payload.IsPanoramic == nil {
	// 	isPanoramic = false
	// } else {
	// 	isPanoramic = *payload.IsPanoramic
	// }

	if payload.IsPublic == nil {
		isPublic = true
	} else {
		isPublic = *payload.IsPublic
	}

	if len(payload.Qualities) == 0 {
		switch payload.Type {
		case models.AudioType:
			payload.Qualities = models.DefaultAudioConfig
		case models.VideoType:
			payload.Qualities = models.DefaultVideoConfig
		}
	} else {
		switch payload.Type {
		case models.AudioType:
			for _, quality := range payload.Qualities {
				if message, ok := quality.IsValid(payload.Type); !ok {
					return response.ResponseFailMessage(ctx, http.StatusBadRequest, message)
				}

				if quality.AudioConfig == nil {
					defaultQualityConfig, ok := models.DefaultConfigMapping[quality.Resolution]
					if !ok {
						return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid audio resolution.")
					}

					quality.AudioConfig = defaultQualityConfig.AudioConfig
				}
			}
		case models.VideoType:
			var haveVideo, shouldAddDefaultHlsAudio, shouldAddDefaultDashAudio bool
			for _, quality := range payload.Qualities {
				if quality.VideoConfig != nil && quality.Resolution == "" {
					return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Quality's resolution is required.")
				}

				if message, ok := quality.IsValid(payload.Type); !ok {
					return response.ResponseFailMessage(ctx, http.StatusBadRequest, message)
				}

				if quality.VideoConfig == nil && quality.AudioConfig == nil {
					if quality.Type == models.HlsQualityType {
						shouldAddDefaultHlsAudio = true
					}

					if quality.Type == models.DashQualityType {
						shouldAddDefaultDashAudio = true
					}

					if quality.Resolution == "" {
						return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Quality's resolution is required.")
					}

					defaultQualityConfig := models.DefaultConfigMapping[quality.Resolution]
					quality.VideoConfig = defaultQualityConfig.VideoConfig
				} else if quality.VideoConfig != nil {
					defaultQualityConfig, ok := models.DefaultConfigMapping[quality.Resolution]
					if !ok {
						return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid quality.")
					}

					quality.VideoConfig.Width = defaultQualityConfig.VideoConfig.Width
					quality.VideoConfig.Height = defaultQualityConfig.VideoConfig.Height
				}

				if !haveVideo && quality.VideoConfig != nil {
					haveVideo = true
				}
			}

			if shouldAddDefaultHlsAudio {
				payload.Qualities = append(
					payload.Qualities,
					&models.QualityConfig{
						Resolution:    "default-audio",
						Type:          models.HlsQualityType,
						ContainerType: models.MpegtsContainerType,
						AudioConfig: &models.AudioConfig{
							Codec:      models.AacCodec,
							Bitrate:    128_000,
							Index:      0,
							SampleRate: 44_100,
						},
					},
				)
			}

			if shouldAddDefaultDashAudio {
				payload.Qualities = append(
					payload.Qualities,
					&models.QualityConfig{
						Resolution:    "default-audio",
						Type:          models.DashQualityType,
						ContainerType: models.Mp4ContainerType,
						AudioConfig: &models.AudioConfig{
							Codec:      models.AacCodec,
							Bitrate:    128_000,
							Index:      0,
							SampleRate: 44_100,
						},
					},
				)
			}

			if !haveVideo {
				return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Video quality is required.")
			}
		}
	}

	if payload.SegmentDuration == 0 ||
		payload.SegmentDuration > models.MaxSegmentDuration {
		payload.SegmentDuration = models.DefaultSegmentDuration
	}

	if len(payload.Tags) != 0 {
		if len(payload.Tags) > models.TagMaxItems {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					nil,
					fmt.Sprintf(
						"Number of tags must be less than %d.",
						models.TagMaxItems,
					),
				),
			)
		}

		filteredTags := make(
			[]string,
			0,
			len(payload.Tags),
		)
		mapTags := make(map[string]bool)
		for _, tag := range payload.Tags {
			tag = strings.TrimSpace(strings.ToLower(tag))
			if len(tag) > models.TagMaxLen {
				return response.ResponseError(
					ctx,
					response.NewHttpError(
						http.StatusBadRequest,
						nil,
						fmt.Sprintf(
							"Tag length must be less than %d characters.",
							models.TagMaxLen,
						),
					),
				)
			}

			if _, ok := mapTags[tag]; !ok {
				filteredTags = append(
					filteredTags,
					tag,
				)
			}
			mapTags[tag] = true
		}

		slices.Sort(filteredTags)
		payload.Tags = filteredTags
	}

	if len(payload.Metadata) != 0 {
		if len(payload.Metadata) > models.MetadataMaxItems {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					nil,
					fmt.Sprintf(
						"Number of metadata must be less than %d.",
						models.MetadataMaxItems,
					),
				),
			)
		}

		filteredMetadata := make(
			[]models.Metadata,
			0,
			len(payload.Metadata),
		)
		mapMetadata := make(map[string]bool)
		for _, meta := range payload.Metadata {
			meta.Key = strings.TrimSpace(strings.ToLower(meta.Key))
			meta.Value = strings.TrimSpace(strings.ToLower(meta.Value))
			if len(meta.Key) > models.MetadataMaxLen ||
				len(meta.Value) > models.MetadataMaxLen {
				return response.ResponseError(
					ctx,
					response.NewHttpError(
						http.StatusBadRequest,
						nil,
						fmt.Sprintf(
							"Metadata key length must be less than %d characters.",
							models.MetadataMaxLen,
						),
					),
				)
			}

			if _, ok := mapMetadata[meta.Key]; !ok {
				filteredMetadata = append(
					filteredMetadata,
					meta,
				)
			}
			mapMetadata[meta.Key] = true
		}

		payload.Metadata = filteredMetadata
	}

	newMedia, err := models.NewMedia(
		authInfo.User.Id,
		payload.Type,
		payload.Title,
		payload.Description,
		payload.Metadata,
		payload.Qualities,
		payload.SegmentDuration,
		payload.Tags,
		isPublic,
	)

	if payload.Watermark != nil {
		watermarkId, err := uuid.Parse(payload.Watermark.Id)
		if err != nil {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					err,
					"Invalid watermark id.",
				),
			)
		}

		newMedia.Watermark = models.NewMediaWatermark(
			newMedia.Id,
			watermarkId,
			payload.Watermark.Width,
			payload.Watermark.Height,
			payload.Watermark.Top,
			payload.Watermark.Left,
			payload.Watermark.Bottom,
			payload.Watermark.Right,
			payload.Watermark.Opacity,
		)

		if !newMedia.Watermark.IsValid() {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					nil,
					"Invalid watermark.",
				),
			)
		}
	}

	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				err.Error(),
			),
		)
	}

	media, err := c.mediaService.CreateMediaObject(
		ctx.Request().Context(),
		newMedia,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewInternalServerError(err),
		)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		models.NewMediaObject(media),
	)
}

type UploadPartRequest struct {
	// Index of the part
	Index int `json:"index" form:"index"`
	// Md5 hash of the part
	Hash string `json:"hash"  form:"hash"`
}

type UploadMediaResponse struct {
	Id uuid.UUID `json:"id"`
}

// UploadMediaThumbnail  godoc
//
//	@Summary			Upload media thumbnail
//	@Tags				media
//	@Id					POST-media-thumbnail
//	@Accept				multipart/form-data
//	@Security			BasicAuth
//	@Security			Bearer
//	@Param				id		path		string	true	"media's id"
//	@Param				file	formData	file	true	"file media to be uploaded"
//	@Success			200		{object}	models.ResponseSuccess
//	@Header				200		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				200		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				200		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			400		{object}	models.ResponseError
//	@Header				400		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			403		{object}	models.ResponseError
//	@Header				403		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				403		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				403		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			404		{object}	models.ResponseError
//	@Header				404		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				404		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				404		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500		{object}	models.ResponseError
//	@Header				500		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router				/media/{id}/thumbnail [post]
//	@x-client-action	"uploadThumbnail"
func (c *MediaController) UploadMediaThumbnail(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UploadMediaThumbnail").
			Observe(time.Since(t).Seconds())
	}()

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media id.",
			),
		)
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid input.",
			),
		)
	}

	if fileHeader.Size > models.MaxThumbnailSize {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				"Thumbnail's size is too large. (max size: 8MB)",
			),
		)
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid input.",
			),
		)
	}

	file, ok := form.File["file"]
	if !ok || len(file) == 0 {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid input.",
			),
		)
	}

	src, err := file[0].Open()
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid input.",
			),
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	if err := c.mediaService.UploadThumbnail(
		ctx.Request().Context(),
		mediaId,
		src,
		authInfo,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Upload thumbnail successfully.",
	)
}

// DeleteMediaThumbnail godoc
//
//	@Summary			Delete media thumbnail
//	@Tags				media
//	@Id					DELETE-media-thumbnail
//	@Security			BasicAuth
//	@Security			Bearer
//	@Param				id	path		string	true	"media's id"
//	@Success			200	{object}	models.ResponseSuccess
//	@Header				200	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				200	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				200	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			400	{object}	models.ResponseError
//	@Header				400	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			403	{object}	models.ResponseError
//	@Header				403	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				403	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				403	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			404	{object}	models.ResponseError
//	@Header				404	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				404	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				404	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500	{object}	models.ResponseError
//	@Header				500	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router				/media/{id}/thumbnail [delete]
//	@x-client-action	"deleteThumbnail"
func (c *MediaController) DeleteMediaThumbnail(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteMediaThumbnail").
			Observe(time.Since(t).Seconds())
	}()

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Invalid media id.",
		)
	}

	authInfo, ok := ctx.Get("authInfo").(models.AuthenticationInfo)
	if !ok {
		return response.ResponseFailMessage(
			ctx,
			http.StatusUnauthorized,
			"Unauthorized.",
		)
	}
	if err := c.mediaService.DeleteMediaThumbnail(
		ctx.Request().Context(),
		mediaId,
		authInfo,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		nil,
	)
}

type UploadMediaPartRequest struct {
	Index string                `json:"index"`                                                                                                      // Index of the part
	Hash  string                `json:"hash"`                                                                                                       // Md5 hash of part
	File  *multipart.FileHeader `json:"file"  swaggertype:"primitive,string" format:"binary" binding:"required" extensions:"x-client-chunk-upload"` // File media to be uploaded
} //	@name	UploadMediaPartRequest

// UploadPart godoc
//
//	@Summary				Upload part of media
//	@Description			Upload part of media
//	@Tags					media
//	@Id						POST-media-part
//	@Accept					multipart/form-data
//	@Security				BasicAuth
//	@Security				Bearer
//	@Param					Content-Range	header		string					false	"file's info"	extensions(x-client-ignore)
//	@Param					id				path		string					true	"media's id"
//	@Param					request			body		UploadMediaPartRequest	true	"part's index"
//	@Success				200				{object}	models.ResponseSuccess
//	@Header					200				{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					200				{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					200				{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure				400				{object}	models.ResponseError
//	@Header					400				{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					400				{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					400				{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure				403				{object}	models.ResponseError
//	@Header					403				{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					403				{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					403				{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure				404				{object}	models.ResponseError
//	@Header					404				{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					404				{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					404				{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure				500				{object}	models.ResponseError
//	@Header					500				{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					500				{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					500				{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router					/media/{id}/part [post]
//	@x-client-action		"uploadPart"
//	@x-client-chunk-upload	true
func (c *MediaController) UploadPart(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UploadPart").
			Observe(time.Since(t).Seconds())
	}()

	var payload UploadPartRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				"Invalid input.",
			),
		)
	}

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media id.",
			),
		)
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid input.",
			),
		)
	}

	file, ok := form.File["file"]
	if !ok || len(file) == 0 {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid input.",
			),
		)
	}

	src, err := file[0].Open()
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid input.",
			),
		)
	}

	hasher := md5.New()
	if _, err := io.Copy(
		hasher,
		src,
	); err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid input.",
			),
		)
	}

	fileHash := fmt.Sprintf("%x", hasher.Sum(nil))
	if fileHash != payload.Hash {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"File corrupted.",
			),
		)
	}

	contentRange := ctx.Request().Header.Get("Content-Range")
	if contentRange == "" {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid content range.",
			),
		)
	}

	re := regexp.MustCompile(`^bytes (\d+)-(\d+)/(\d+)$`)
	matches := re.FindStringSubmatch(contentRange)
	if len(matches) != 4 {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid content range.",
			),
		)
	}

	startPos, err := strconv.ParseInt(
		matches[1],
		10,
		64,
	)
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid content range.",
			),
		)
	}

	endPos, err := strconv.ParseInt(
		matches[2],
		10,
		64,
	)
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid content range.",
			),
		)
	}

	fileSize, err := strconv.ParseInt(
		matches[3],
		10,
		64,
	)
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid content range.",
			),
		)
	}

	if endPos-startPos+1 != file[0].Size {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				"Invalid content range.",
			),
		)
	}

	if fileSize > models.MaxMediaSize {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				"Media's size is too large. (max size: 2TB)",
			),
		)
	}

	authInfo, ok := ctx.Get("authInfo").(models.AuthenticationInfo)
	if !ok {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusUnauthorized,
				nil,
				"Unauthorized.",
			),
		)
	}

	part := models.NewPart(
		mediaId,
		authInfo.User.Id,
		payload.Hash,
		payload.Index,
		endPos-startPos+1,
	)
	if err := c.mediaService.UploadPart(
		ctx.Request().Context(),
		mediaId,
		fileSize,
		part,
		src,
		authInfo,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Upload part successfully.",
	)
}

// UploadMediaComplete godoc
//
//	@Summary			Get upload media when complete
//	@Description		Get upload media when complete.
//	@Id					GET-media-complete
//	@Tags				media
//	@Security			BasicAuth
//	@Security			Bearer
//	@Param				id	path		string	true	"media's id"
//	@Success			200	{object}	models.ResponseSuccess
//	@Header				200	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				200	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				200	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			400	{object}	models.ResponseError
//	@Header				400	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			403	{object}	models.ResponseError
//	@Header				403	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				403	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				403	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			404	{object}	models.ResponseError
//	@Header				404	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				404	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				404	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500	{object}	models.ResponseError
//	@Header				500	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router				/media/{id}/complete [get]
//	@x-client-action	"uploadMediaComplete"
func (c *MediaController) UploadMediaComplete(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UploadMediaComplete").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media id.",
			),
		)
	}

	if err := c.mediaService.UploadMediaComplete(
		ctx.Request().Context(),
		mediaId,
		authInfo,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Upload media complete successfully.",
	)
}

type GetMediaListRequest struct {
	Offset   int               `json:"offset"   form:"offset"`
	Limit    int               `json:"limit"    form:"limit"`
	SortBy   string            `json:"sort_by"  form:"sort_by"`
	OrderBy  string            `json:"order_by" form:"order_by"`
	Type     string            `json:"type"     form:"type"`
	Status   []string          `json:"status"   form:"status"`
	Search   string            `json:"search"   form:"search"`
	Metadata []models.Metadata `json:"metadata" form:"metadata"`
	Tags     []string          `json:"tags"     form:"tags"`
} //	@name	GetMediaListRequest

type GetMediaListData struct {
	Medias []*models.MediaObject `json:"media"`
	Total  int64                 `json:"total"`
} //	@name	GetMediaListData

type GetMediaListResponse struct {
	Status string           `json:"status"`
	Data   GetMediaListData `json:"data"`
} //	@name	GetMediaListResponse

// GetMediaList godoc
//
//	@Summary			Get user media list
//	@Description		Retrieve a list of media for the authenticated user.
//	@Id					POST-list-media
//	@Tags				media
//	@Accept				json
//	@Produce			json
//	@Security			BasicAuth
//	@Security			Bearer
//	@Param				request	body		GetMediaListRequest		true	"media's info"
//	@Header				200		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Success			200		{object}	GetMediaListResponse	"Successful response containing the media list"
//	@Header				200		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				200		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			400		{object}	models.ResponseError
//	@Header				400		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			403		{object}	models.ResponseError
//	@Header				403		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				403		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				403		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			404		{object}	models.ResponseError
//	@Header				404		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				404		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				404		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500		{object}	models.ResponseError
//	@Header				500		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router				/media	[post]
//	@x-client-action	"getMediaList"
func (c *MediaController) GetMediaList(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaList").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetMediaListRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid input.",
			),
		)
	}

	if payload.Limit > models.PageSizeLimit {
		payload.Limit = models.PageSizeLimit
	}

	if payload.Limit <= 0 {
		payload.Limit = models.DefaultPageLimit
	}

	if payload.Offset < 0 {
		payload.Offset = 0
	}

	if len(payload.Status) > 0 {
		for _, status := range payload.Status {
			if !slices.Contains(models.ValidMediaStatus, status) {
				return response.ResponseError(
					ctx,
					response.NewHttpError(
						http.StatusBadRequest,
						nil,
						"Invalid status.",
					),
				)
			}
		}
	}

	if _, ok := models.ValidMediaSortByColumns[payload.SortBy]; !ok {
		payload.SortBy = models.DefaultMediaSortBy
	}

	if _, ok := models.OrderMap[payload.OrderBy]; !ok {
		payload.OrderBy = models.DefaultOrderBy
	}

	if payload.Type == "" {
		payload.Type = models.VideoType
	} else if payload.Type != models.VideoType && payload.Type != models.AudioType {
		return response.ResponseFailMessage(ctx, http.StatusBadGateway, "Invalid media type.")
	}

	if len(payload.Tags) != 0 {
		if len(payload.Tags) > models.TagMaxItems {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					nil,
					fmt.Sprintf(
						"Number of tags must be less than %d.",
						models.TagMaxItems,
					),
				),
			)
		}

		filteredTags := make(
			[]string,
			0,
			len(payload.Tags),
		)
		mapTags := make(map[string]bool)
		for _, tag := range payload.Tags {
			tag = strings.TrimSpace(strings.ToLower(tag))
			if len(tag) > models.TagMaxLen {
				return response.ResponseError(
					ctx,
					response.NewHttpError(
						http.StatusBadRequest,
						nil,
						fmt.Sprintf(
							"Tag length must be less than %d characters.",
							models.TagMaxLen,
						),
					),
				)
			}

			if _, ok := mapTags[tag]; !ok {
				filteredTags = append(
					filteredTags,
					tag,
				)
			}
			mapTags[tag] = true
		}

		slices.Sort(filteredTags)
		payload.Tags = filteredTags
	}

	if len(payload.Metadata) != 0 {
		if len(payload.Metadata) > models.MetadataMaxItems {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					nil,
					fmt.Sprintf(
						"Number of metadata must be less than %d.",
						models.MetadataMaxItems,
					),
				),
			)
		}

		filteredMetadata := make(
			[]models.Metadata,
			0,
			len(payload.Metadata),
		)
		mapMetadata := make(map[string]bool)
		for _, meta := range payload.Metadata {
			meta.Key = strings.TrimSpace(strings.ToLower(meta.Key))
			meta.Value = strings.TrimSpace(strings.ToLower(meta.Value))
			if len(meta.Key) > models.MetadataMaxLen ||
				len(meta.Value) > models.MetadataMaxLen {
				return response.ResponseError(
					ctx,
					response.NewHttpError(
						http.StatusBadRequest,
						nil,
						fmt.Sprintf(
							"Metadata key length must be less than %d characters.",
							models.MetadataMaxLen,
						),
					),
				)
			}

			if _, ok := mapMetadata[meta.Key]; !ok {
				filteredMetadata = append(
					filteredMetadata,
					meta,
				)
			}

			mapMetadata[meta.Key] = true
		}

		payload.Metadata = filteredMetadata
	}

	if payload.Search != "" {
		payload.Search = strings.TrimSpace(payload.Search)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	media, total, err := c.mediaService.GetMediaList(
		ctx.Request().Context(),
		models.GetMediaListInput{
			OrderBy:  payload.OrderBy,
			SortBy:   payload.SortBy,
			Status:   payload.Status,
			Offset:   payload.Offset,
			Limit:    payload.Limit,
			Metadata: payload.Metadata,
			Tags:     payload.Tags,
			Search:   payload.Search,
			Type:     payload.Type,
		},
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		GetMediaListData{
			Medias: models.NewMediaObjects(media),
			Total:  total,
		},
	)
}

type GetTranscodeCostData struct {
	Price    float64 `json:"price"     swaggertype:"string"`
	IsEnough bool    `json:"is_enough"`
} //	@name	GetTranscodeCostData
type GetTranscodeCostResponse struct {
	Status string               `json:"status"`
	Data   GetTranscodeCostData `json:"data"`
} //	@name	GetTranscodeCostResponse

// GetTranscodeCost godoc
//
//	@Summary			get media transcoding cost
//	@Description		get media transcoding cost
//	@Id					GET-transcode-cost
//	@Tags				media
//	@Security			BasicAuth
//	@Security			Bearer
//	@Param				qualities	query		string	true	"media's qualities"
//	@Param				type		query		string	true	"media's type"
//	@Param				duration	query		float64	true	"media's duration"
//	@Success			200			{object}	GetTranscodeCostResponse
//	@Header				200			{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				200			{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				200			{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			400			{object}	models.ResponseError
//	@Header				400			{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400			{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400			{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			403			{object}	models.ResponseError
//	@Header				403			{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				403			{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				403			{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			404			{object}	models.ResponseError
//	@Header				404			{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				404			{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				404			{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500			{object}	models.ResponseError
//	@Header				500			{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500			{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500			{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router	/media/cost	[get]
//	@x-client-action	"getCost"
func (c *MediaController) GetTranscodeCost(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetTranscodeCost").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	mediaType := ctx.QueryParam("type")
	if mediaType == "" {
		mediaType = models.VideoType
	}

	qualities := strings.Split(
		ctx.QueryParam("qualities"),
		",",
	)
	for _, quality := range qualities {
		if _, ok := models.ValidMediaQualities[quality]; !ok && mediaType == models.VideoType {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					nil,
					"Invalid quality.",
				),
			)
		}
	}

	duration, err := strconv.ParseFloat(
		ctx.QueryParam("duration"),
		64,
	)
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid duration.",
			),
		)
	}

	if duration <= 0 {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				"Duration must be greater than 0.",
			),
		)
	}

	if math.IsInf(duration, 1) || math.IsInf(duration, -1) {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Invalid duration.",
		)
	}

	price, isEnough, err := c.mediaService.CalculateMediaCost(
		ctx.Request().Context(),
		duration,
		mediaType,
		qualities,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		GetTranscodeCostData{
			Price:    price.InexactFloat64(),
			IsEnough: isEnough,
		},
	)
}

type GetMediaDetailResponse struct {
	Status string              `json:"status"`
	Data   *models.MediaObject `json:"data"`
} //	@name	GetMediaDetailResponse

// GetMediaDetail godoc
//
//	@Summary			get media detail
//	@Description		Retrieve the media details by media id.
//	@Id					GET-media-status
//	@Tags				media
//	@Produce			json
//	@Security			BasicAuth
//	@Security			Bearer
//	@Param				id	path		string	true	"media's id"
//	@Success			200	{object}	GetMediaDetailResponse
//	@Header				200	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				200	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				200	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			400	{object}	models.ResponseError
//	@Header				400	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			403	{object}	models.ResponseError
//	@Header				403	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				403	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				403	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			404	{object}	models.ResponseError
//	@Header				404	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				404	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				404	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500	{object}	models.ResponseError
//	@Header				500	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router				/media/{id} [get]
//	@x-client-action	"getDetail"
func (c *MediaController) GetMediaDetail(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaDetail").
			Observe(time.Since(t).Seconds())
	}()

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media id.",
			),
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	media, err := c.mediaService.GetMediaDetail(
		ctx.Request().Context(),
		mediaId,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		models.NewMediaObject(media),
	)
}

// Get media manifest
//
//	@Summary	get media's dash manifest file
//	@tags		media
//	@Security	jwt
//	@Param		media's	id			path	string	true	"media's id"
//	@Success	200		{object}	string
//	@Failure	400		{object}	models.ResponseError
//	@Failure	404		{object}	models.ResponseError
//	@Router		/media/{id}/manifest.mpd [get]
func (c *MediaController) GetMediaDashManifest(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaManifest").
			Observe(time.Since(t).Seconds())
	}()

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media id.",
			),
		)
	}

	fileInfo, err := c.mediaService.GetMediaDashManifest(
		ctx.Request().Context(),
		mediaId,
		ctx.QueryParam("token"),
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	defer func() {
		if fileInfo.Reader != nil {
			if closer, ok := fileInfo.Reader.(io.Closer); ok {
				closer.Close()
			}
		}

		if err := c.usageService.CreateUsageLog(
			ctx.Request().Context(),
			(&models.UsageLogBuilder{}).SetUserId(fileInfo.UserId).
				SetDelivery(fileInfo.Size).
				SetIsUserCost(false).
				Build(),
		); err != nil {
			slog.ErrorContext(
				ctx.Request().Context(),
				"Create usage log error",
				slog.Any("err", err),
			)
		}
	}()

	ctx.Response().
		Header().
		Set("Content-Disposition", "attachment; filename=manifest.mpd")
	ctx.Response().
		Header().
		Set("Content-Length", strconv.FormatInt(fileInfo.Size, 10))
	ctx.Response().Header().Set(
		"Content-Type",
		"application/dash+xml",
	)
	ctx.Response().WriteHeader(http.StatusOK)
	n, err := io.Copy(ctx.Response(), fileInfo.Reader)
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewInternalServerError(err),
		)
	}

	fileInfo.Size = n

	return nil
}

// Get media m3u8
//
//	@Summary	get media's hls manifest file
//	@tags		media
//	@Security	jwt
//	@Param		media's	id			path	string	true	"media's id"
//	@Success	200		{object}	string
//	@Failure	400		{object}	models.ResponseError
//	@Failure	404		{object}	models.ResponseError
//	@Router		/media/{id}/manifest.m3u8 [get]
func (c *MediaController) GetMediaHlsManifest(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaM3U8").
			Observe(time.Since(t).Seconds())
	}()

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media id.",
			),
		)
	}

	fileInfo, err := c.mediaService.GetMediaM3U8(
		ctx.Request().Context(),
		mediaId,
		ctx.QueryParam("token"),
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	defer func() {
		if fileInfo.Reader != nil {
			if closer, ok := fileInfo.Reader.(io.Closer); ok {
				closer.Close()
			}
		}

		if err := c.usageService.CreateUsageLog(
			ctx.Request().Context(),
			(&models.UsageLogBuilder{}).SetUserId(fileInfo.UserId).
				SetDelivery(fileInfo.Size).
				SetIsUserCost(false).
				Build(),
		); err != nil {
			slog.ErrorContext(
				ctx.Request().Context(),
				"Create usage log error",
				slog.Any("err", err),
			)
		}
	}()

	ctx.Response().
		Header().
		Set("Content-Disposition", "attachment; filename=master.m3u8")
	ctx.Response().
		Header().
		Set("Content-Length", strconv.FormatInt(fileInfo.Size, 10))
	ctx.Response().Header().Set(
		"Content-Type",
		"application/vnd.apple.mpegurl",
	)
	ctx.Response().WriteHeader(http.StatusOK)
	n, err := io.Copy(ctx.Response(), fileInfo.Reader)
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewInternalServerError(err),
		)
	}

	fileInfo.Size = n

	return nil
}

// Get media mp4
//
//	@Summary	get media's mp4 file
//	@tags		media
//	@Security	jwt
//	@Param		media's	id			path	string	true	"media's id"
//	@Success	200		{object}	string
//	@Failure	400		{object}	models.ResponseError
//	@Failure	404		{object}	models.ResponseError
//	@Router		/media/{id}/mp4 [get]
func (c *MediaController) GetMediaMp4(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaMp4").
			Observe(time.Since(t).Seconds())
	}()

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media id.",
			),
		)
	}

	if err := c.mediaService.GetMediaMp4(
		ctx,
		mediaId,
		ctx.QueryParam("token"),
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return nil
}

func (c *MediaController) GetMediaAudio(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaAudio").
			Observe(time.Since(t).Seconds())
	}()

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media id.",
			),
		)
	}

	reader, err := c.mediaService.GetMediaAudio(
		ctx.Request().Context(),
		mediaId,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	defer reader.Close()
	if err := ctx.Stream(http.StatusOK, "audio/mp3", reader); err != nil {
		return response.ResponseError(
			ctx,
			response.NewInternalServerError(err),
		)
	}

	return nil
}

// Get media thumbnail
//
//	@Summary	get media's thumbnail image
//	@tags		media
//	@Security	jwt
//	@Param		media's		id			path	string	true	"media's id"
//	@Param		resolution	query		string	true	"image resolution"
//	@Success	200			{object}	string
//	@Failure	400			{object}	models.ResponseError
//	@Failure	404			{object}	models.ResponseError
//	@Router		/media/{id}/thumbnail [get]
func (c *MediaController) GetMediaThumbnail(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaThumbnail").
			Observe(time.Since(t).Seconds())
	}()

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media id.",
			),
		)
	}

	resolution := ctx.QueryParam("resolution")
	if resolution == "" {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				"Invalid resolution.",
			),
		)
	}

	fileInfo, err := c.mediaService.GetMediaThumbnail(
		ctx.Request().Context(),
		mediaId,
		resolution,
		ctx.QueryParam("token"),
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	defer func() {
		if fileInfo.Reader != nil {
			if closer, ok := fileInfo.Reader.(io.Closer); ok {
				closer.Close()
			}
		}

		if err := c.usageService.CreateUsageLog(
			ctx.Request().Context(),
			(&models.UsageLogBuilder{}).SetUserId(fileInfo.UserId).
				SetDelivery(fileInfo.Size).
				SetIsUserCost(false).
				Build(),
		); err != nil {
			slog.ErrorContext(
				ctx.Request().Context(),
				"Create usage log error",
				slog.Any("err", err),
			)
		}
	}()

	if fileInfo.RedirectUrl != "" {
		expiredAt := time.Unix(0, fileInfo.ExpiredAt)
		ctx.Response().
			Header().
			Set("Cahche-Control", fmt.Sprintf("max-age=%d", int(time.Until(expiredAt))))
		ctx.Response().Header().Set(
			"Expires",
			time.Unix(0, fileInfo.ExpiredAt).Format(http.TimeFormat),
		)
		return ctx.Redirect(http.StatusTemporaryRedirect, fileInfo.RedirectUrl)
	}

	ctx.Response().Header().Set(
		"Content-Type",
		"image/*",
	)
	ctx.Response().WriteHeader(http.StatusOK)
	n, err := io.Copy(ctx.Response(), fileInfo.Reader)
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewInternalServerError(err),
		)
	}

	fileInfo.Size = n

	return nil
}

func (c *MediaController) GetMediaChapter(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaChapter").
			Observe(time.Since(t).Seconds())
	}()

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media's id.",
			),
		)
	}

	lan := strings.TrimSuffix(ctx.Param("lan"), ".vtt")

	if _, ok := models.ValidLanguageMapping[lan]; !ok {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				"Invalid language.",
			),
		)
	}

	fileInfo, err := c.mediaService.GetMediaChapter(
		ctx.Request().Context(),
		mediaId,
		lan,
		ctx.QueryParam("token"),
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	defer func() {
		if fileInfo.Reader != nil {
			if closer, ok := fileInfo.Reader.(io.Closer); ok {
				closer.Close()
			}
		}

		if err := c.usageService.CreateUsageLog(
			ctx.Request().Context(),
			(&models.UsageLogBuilder{}).SetUserId(fileInfo.UserId).
				SetDelivery(fileInfo.Size).
				SetIsUserCost(false).
				Build(),
		); err != nil {
			slog.ErrorContext(
				ctx.Request().Context(),
				"Create usage log error",
				slog.Any("err", err),
			)
		}
	}()

	if fileInfo.RedirectUrl != "" {
		expiredAt := time.Unix(0, fileInfo.ExpiredAt)
		ctx.Response().
			Header().
			Set("Cahche-Control", fmt.Sprintf("max-age=%d", int(time.Until(expiredAt))))
		ctx.Response().Header().Set(
			"Expires",
			time.Unix(0, fileInfo.ExpiredAt).Format(http.TimeFormat),
		)
		return ctx.Redirect(http.StatusTemporaryRedirect, fileInfo.RedirectUrl)
	}

	ctx.Response().
		Header().
		Set("Content-Disposition", fmt.Sprintf("attachment; filename=chapter_%s.vtt", ctx.Param("lan")))
	ctx.Response().Header().Set("Content-Type", "text/vtt")
	ctx.Response().WriteHeader(http.StatusOK)
	n, err := io.Copy(ctx.Response(), fileInfo.Reader)
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewInternalServerError(err),
		)
	}

	fileInfo.Size = n

	return nil
}

func (c *MediaController) GetMediaCaption(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaCaption").
			Observe(time.Since(t).Seconds())
	}()

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media's id.",
			),
		)
	}

	langParam := ctx.Param("lan")
	re := regexp.MustCompile(`^.{0,2}`)
	lang := re.FindString(langParam)
	if _, ok := models.ValidLanguageMapping[lang]; !ok {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				"Invalid language.",
			),
		)
	}

	fileInfo, err := c.mediaService.GetMediaCaption(
		ctx.Request().Context(),
		mediaId,
		ctx.QueryParam("fileType"),
		ctx.Param("lan"),
		ctx.QueryParam("token"),
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	defer func() {
		if fileInfo.Reader != nil {
			if closer, ok := fileInfo.Reader.(io.Closer); ok {
				closer.Close()
			}
		}

		if err := c.usageService.CreateUsageLog(
			ctx.Request().Context(),
			(&models.UsageLogBuilder{}).SetUserId(fileInfo.UserId).
				SetDelivery(fileInfo.Size).
				SetIsUserCost(false).
				Build(),
		); err != nil {
			slog.ErrorContext(
				ctx.Request().Context(),
				"Create usage log error",
				slog.Any("err", err),
			)
		}
	}()

	if fileInfo.RedirectUrl != "" {
		expiredAt := time.Unix(0, fileInfo.ExpiredAt)
		ctx.Response().
			Header().
			Set("Cahche-Control", fmt.Sprintf("max-age=%d", int(time.Until(expiredAt))))
		ctx.Response().Header().Set(
			"Expires",
			time.Unix(0, fileInfo.ExpiredAt).Format(http.TimeFormat),
		)
		return ctx.Redirect(http.StatusTemporaryRedirect, fileInfo.RedirectUrl)
	}

	fileType := ctx.QueryParam("fileType")
	var mimeType, fileName string
	switch fileType {
	case models.CdnCaptionType:
		mimeType = "text/vtt"
		fileName = "caption.vtt"
	case models.CdnCaptionM3u8Type:
		mimeType = "application/vnd.apple.mpegurl"
		fileName = "caption.m3u8"
	default:
		mimeType = "text/vtt"
	}

	ctx.Response().
		Header().
		Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	ctx.Response().Header().Set("Content-Type", mimeType)
	ctx.Response().WriteHeader(http.StatusOK)
	n, err := io.Copy(ctx.Response(), fileInfo.Reader)
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewInternalServerError(err),
		)
	}

	fileInfo.Size = n

	return nil
}

type GetMediaPlayerInfoResponse struct {
	Id            string                  `json:"id"`
	UserId        string                  `json:"user_id"`
	Title         string                  `json:"title"`
	Description   string                  `json:"description"`
	Metadata      []*models.Metadata      `json:"metadata"`
	Tags          []string                `json:"tags"`
	Qualities     []*models.QualityObject `json:"qualities"`
	Captions      []*models.MediaCaption  `json:"captions"`
	Chapters      []*models.MediaChapter  `json:"chapters"`
	PlayerTheme   *models.PlayerTheme     `json:"player_theme"`
	Duration      float64                 `json:"duration"`
	Size          int64                   `json:"size"`
	IsMp4         bool                    `json:"is_mp4"`
	IsPublic      bool                    `json:"is_public"`
	Status        string                  `json:"status"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
	Assets        *models.MediaAssets     `json:"assets"`
	PlayerThemeId string                  `json:"player_theme_id"`
} //	@name	GetMediaPlayerInfoResponse

// GetMediaObject godoc
//
//	@Summary			Get media player info
//	@Description		Get media player info
//	@Tags				media
//	@Produce			json
//	@Param				id		path		string						true	"Media ID"
//	@Param				token	query		string						false	"Token"
//	@Success			200		{object}	GetMediaPlayerInfoResponse	"MediaObject"
//	@Header				200		{integer}	X-RateLimit-Limit			"The request limit per minute"
//	@Header				200		{integer}	X-RateLimit-Remaining		"The number of available requests left for the current time window"
//	@Header				200		{integer}	X-RateLimit-Retry-After		"The number of seconds left until the current rate limit window resets"
//	@Failure			400		{object}	models.ResponseError
//	@Header				400		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			403		{object}	models.ResponseError
//	@Header				403		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				403		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				403		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			404		{object}	models.ResponseError
//	@Header				404		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				404		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				404		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500		{object}	models.ResponseError
//	@Header				500		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router				/media/{id}/player.json [get]
//	@x-client-action	"getMediaPlayerInfo"
//	@x-group-parameters	true
func (c *MediaController) GetMediaObject(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaObject").
			Observe(time.Since(t).Seconds())
	}()

	var mediaId uuid.UUID
	var err error
	if ctx.Param("id") != "demo" {
		mediaId, err = uuid.Parse(ctx.Param("id"))
		if err != nil {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					err,
					"Invalid media's id.",
				),
			)
		}
	} else {
		mediaId = models.DemoVideoId
	}

	media, err := c.mediaService.GetMediaByIdAndToken(
		ctx.Request().Context(),
		mediaId,
		ctx.QueryParam("token"),
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return ctx.JSON(http.StatusOK, models.NewMediaObject(media))
}

func (c *MediaController) GetMediaContent(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaContent").
			Observe(time.Since(t).Seconds())
	}()

	belongsToId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid id.",
			),
		)
	}

	fileRange := ctx.QueryParam("range")
	rangeData := strings.Split(fileRange, ",")
	if len(rangeData) != 2 {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				"Invalid range.",
			),
		)
	}

	offset, err := strconv.ParseInt(rangeData[0], 10, 64)
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid range.",
			),
		)
	}

	size, err := strconv.ParseInt(rangeData[1], 10, 64)
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid range.",
			),
		)
	}

	var index int
	if ctx.QueryParam("index") == "" {
		index = 1
	} else {
		index, err = strconv.Atoi(ctx.QueryParam("index"))
		if err != nil {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					err,
					"Invalid index.",
				),
			)
		}
	}

	contentType := ctx.QueryParam("type")
	if contentType == "" {
		contentType = "video"
	}

	fileInfo, err := c.mediaService.GetMediaContent(
		ctx.Request().Context(),
		belongsToId,
		offset,
		size,
		index,
		ctx.Param("filename"),
		ctx.QueryParam("type"),
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	defer func() {
		if fileInfo.Reader != nil {
			if closer, ok := fileInfo.Reader.(io.Closer); ok {
				closer.Close()
			}
		}

		if err := c.usageService.CreateUsageLog(
			ctx.Request().Context(),
			(&models.UsageLogBuilder{}).SetUserId(fileInfo.UserId).
				SetDelivery(fileInfo.Size).
				SetIsUserCost(true).
				Build(),
		); err != nil {
			slog.ErrorContext(
				ctx.Request().Context(),
				"Create usage log error",
				slog.Any("err", err),
			)
		}
	}()

	mimeType := "application/octet-stream"
	if fileInfo.MimeType != "" {
		mimeType = fileInfo.MimeType
	}

	if fileInfo.RedirectUrl != "" {
		expiredAt := time.Unix(0, fileInfo.ExpiredAt)
		ctx.Response().
			Header().
			Set("Cahche-Control", fmt.Sprintf("max-age=%d", int(time.Until(expiredAt))))
		ctx.Response().Header().Set(
			"Expires",
			time.Unix(0, fileInfo.ExpiredAt).Format(http.TimeFormat),
		)
		return ctx.Redirect(http.StatusTemporaryRedirect, fileInfo.RedirectUrl)
	}

	fileInfo.Size = size
	return ctx.Stream(http.StatusOK, mimeType, fileInfo.Reader)
}

// Get media source
//
//	@Summary	get media's source
//	@tags		media
//	@Security	jwt
//	@Param		media's			id			path	string	true	"media's id"
//	@Param		vms-api-key		header		string	false	"api key (must be select upload token or api key)"
//	@Param		vms-api-secret	header		string	false	"api secret (must be select upload token or api key)"
//	@Param		Authorization	header		string	false	"authorization"
//	@Success	200				{object}	string
//	@Failure	400				{object}	models.ResponseError
//	@Failure	404				{object}	models.ResponseError
//	@Router		/media/{id}/source [get]
func (c *MediaController) GetMediaSource(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaSource").
			Observe(time.Since(t).Seconds())
	}()

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media id.",
			),
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	totalBytes, err := c.mediaService.GetMediaSource(
		ctx,
		mediaId,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	if err := c.usageService.CreateUsageLog(
		ctx.Request().Context(),
		(&models.UsageLogBuilder{}).SetUserId(authInfo.User.Id).
			SetDelivery(totalBytes).
			SetIsUserCost(true).
			Build(),
	); err != nil {
		slog.ErrorContext(
			ctx.Request().Context(),
			"Create usage log error",
			slog.Any("err", err),
		)
	}

	return nil
}

type UpdateMediaInfoRequest struct {
	// Title of the media
	Title *string `json:"title"       form:"title"`
	// Description of the media
	Description *string `json:"description" form:"description"`
	// Media's tags
	Tags []string `json:"tags"        form:"tags"`
	// Media's metadata
	Metadata []models.Metadata `json:"metadata"    form:"metadata"`
	// Media player 's id
	PlayerId *string `json:"player_id"   form:"player_id"`
	// Media's publish status
	IsPublic *bool `json:"is_public"   form:"is_public"`
} //	@name	UpdateMediaInfoRequest

// UpdateMediaInfo godoc
//
//	@Summary			update media info
//	@Tags				media
//	@Id					PATCH_media
//	@Security			BasicAuth
//	@Security			Bearer
//	@Accept				json
//	@Produce			json
//	@Param				id		path		string					true	"media's id"
//	@Param				input	body		UpdateMediaInfoRequest	true	"input"
//	@Success			200		{object}	models.ResponseSuccess
//	@Header				200		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				200		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				200		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			400		{object}	models.ResponseError
//	@Header				400		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			403		{object}	models.ResponseError
//	@Header				403		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				403		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				403		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			404		{object}	models.ResponseError
//	@Header				404		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				404		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				404		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500		{object}	models.ResponseError
//	@Header				500		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router				/media/{id} [patch]
//	@x-client-action	"update"
func (c *MediaController) UpdateMediaInfo(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UpdateMediaInfo").
			Observe(time.Since(t).Seconds())
	}()

	var payload UpdateMediaInfoRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid input.",
			),
		)
	}

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media id.",
			),
		)
	}

	if payload.Title != nil {
		trimTitle := strings.TrimSpace(*payload.Title)
		if trimTitle == "" {
			return response.ResponseFailMessage(
				ctx,
				http.StatusBadRequest,
				"Title is required.",
			)
		}
		payload.Title = &trimTitle
		if len(*payload.Title) > models.TitleMaxLen {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					nil,
					fmt.Sprintf(
						"Title length must be less than %d characters.",
						models.TitleMaxLen,
					),
				),
			)
		}
	}

	if payload.Description != nil {
		trimDescription := strings.TrimSpace(*payload.Description)
		payload.Description = &trimDescription
		if len(*payload.Description) > models.DescriptionMaxLen {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					nil,
					fmt.Sprintf(
						"Description length must be less than %d characters.",
						models.DescriptionMaxLen,
					),
				),
			)
		}
	}

	if len(payload.Tags) != 0 {
		if len(payload.Tags) > models.TagMaxItems {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					nil,
					fmt.Sprintf(
						"Number of tags must be less than %d.",
						models.TagMaxItems,
					),
				),
			)
		}

		filteredTags := make(
			[]string,
			0,
			len(payload.Tags),
		)
		mapTags := make(map[string]bool)
		for _, tag := range payload.Tags {
			tag = strings.TrimSpace(strings.ToLower(tag))
			if len(tag) > models.TagMaxLen {
				return response.ResponseError(
					ctx,
					response.NewHttpError(
						http.StatusBadRequest,
						nil,
						fmt.Sprintf(
							"Tag length must be less than %d characters.",
							models.TagMaxLen,
						),
					),
				)
			}

			if _, ok := mapTags[tag]; !ok {
				filteredTags = append(
					filteredTags,
					tag,
				)
			}
			mapTags[tag] = true
		}

		slices.Sort(filteredTags)
		payload.Tags = filteredTags
	}

	if len(payload.Metadata) != 0 {
		if len(payload.Metadata) > models.MetadataMaxItems {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					nil,
					fmt.Sprintf(
						"Number of metadata must be less than %d.",
						models.MetadataMaxItems,
					),
				),
			)
		}

		filteredMetadata := make(
			[]models.Metadata,
			0,
			len(payload.Metadata),
		)
		mapMetadata := make(map[string]bool)
		for _, meta := range payload.Metadata {
			meta.Key = strings.TrimSpace(strings.ToLower(meta.Key))
			meta.Value = strings.TrimSpace(strings.ToLower(meta.Value))
			if len(meta.Key) > models.MetadataMaxLen ||
				len(meta.Value) > models.MetadataMaxLen {
				return response.ResponseError(
					ctx,
					response.NewHttpError(
						http.StatusBadRequest,
						nil,
						fmt.Sprintf(
							"Metadata key length must be less than %d characters.",
							models.MetadataMaxLen,
						),
					),
				)
			}

			if _, ok := mapMetadata[meta.Key]; !ok {
				filteredMetadata = append(
					filteredMetadata,
					meta,
				)
			}
			mapMetadata[meta.Key] = true
		}

		payload.Metadata = filteredMetadata
	}

	var playerId *uuid.UUID
	if payload.PlayerId != nil {
		id, err := uuid.Parse(*payload.PlayerId)
		if err != nil {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					err,
					"Invalid player id.",
				),
			)
		}

		playerId = &id
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	if err := c.mediaService.UpdateMediaInfo(
		ctx.Request().Context(),
		mediaId,
		models.UpdateMediaInfoInput{
			Title:       payload.Title,
			Description: payload.Description,
			Tags:        payload.Tags,
			Metadata:    payload.Metadata,
			PlayerId:    playerId,
			IsPublic:    payload.IsPublic,
		},
		authInfo,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Update media info successfully.",
	)
}

type UpdateMediaSettingRequest struct {
	IsPublic bool `json:"is_public" form:"is_public"`
} //	@name	UpdateMediaSettingRequest

// DeleteMedia godoc
//
//	@Summary			Delete media
//	@Description		Delete a media by media ID.
//	@Tags				media
//	@Id					DELETE_media-mediaId
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id	path		string					true	"Media ID"
//	@Success			200	{object}	models.ResponseSuccess	"Delete media successfully."
//	@Failure			400	{object}	models.ResponseError
//	@Failure			403	{object}	models.ResponseError
//	@Failure			404	{object}	models.ResponseError
//	@Failure			500	{object}	models.ResponseError
//	@Header				200	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				200	{integer}	X-RateLimit-Remaining	"Available requests left for current window"
//	@Header				200	{integer}	X-RateLimit-Retry-After	"Seconds until rate limit window resets"
//	@Header				400	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400	{integer}	X-RateLimit-Remaining	"Available requests left for current window"
//	@Header				400	{integer}	X-RateLimit-Retry-After	"Seconds until rate limit window resets"
//	@Header				403	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				403	{integer}	X-RateLimit-Remaining	"Available requests left for current window"
//	@Header				403	{integer}	X-RateLimit-Retry-After	"Seconds until rate limit window resets"
//	@Header				404	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				404	{integer}	X-RateLimit-Remaining	"Available requests left for current window"
//	@Header				404	{integer}	X-RateLimit-Retry-After	"Seconds until rate limit window resets"
//	@Header				500	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500	{integer}	X-RateLimit-Remaining	"Available requests left for current window"
//	@Header				500	{integer}	X-RateLimit-Retry-After	"Seconds until rate limit window resets"
//	@Router				/media/{id} [delete]
//	@x-client-action	"delete"
func (c *MediaController) DeleteMedia(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteMedia").
			Observe(time.Since(t).Seconds())
	}()

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				err,
				"Invalid media id.",
			),
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	if err := c.mediaService.DeleteMedia(
		ctx.Request().Context(),
		mediaId,
		authInfo,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Media deleted successfully.",
	)
}

func (c *MediaController) GetDemoManifest(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaM3U8").
			Observe(time.Since(t).Seconds())
	}()

	mediaType := ctx.Param("id")
	if mediaType != models.HlsQualityType &&
		mediaType != models.DashQualityType {
		return response.ResponseError(
			ctx,
			response.NewHttpError(
				http.StatusBadRequest,
				nil,
				"Invalid media type.",
			),
		)
	}

	fileInfo, err := c.mediaService.GetDemoManifest(
		ctx.Request().Context(),
		mediaType,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	defer func() {
		if fileInfo.Reader != nil {
			if closer, ok := fileInfo.Reader.(io.Closer); ok {
				closer.Close()
			}
		}

		if err := c.usageService.CreateUsageLog(
			ctx.Request().Context(),
			(&models.UsageLogBuilder{}).SetUserId(fileInfo.UserId).
				SetDelivery(fileInfo.Size).
				SetIsUserCost(false).
				Build(),
		); err != nil {
			slog.ErrorContext(
				ctx.Request().Context(),
				"Create usage log error",
				slog.Any("err", err),
			)
		}
	}()

	ctx.Response().
		Header().
		Set("Content-Disposition", "attachment; filename=master.m3u8")
	ctx.Response().
		Header().
		Set("Content-Length", strconv.FormatInt(fileInfo.Size, 10))
	ctx.Response().Header().Set(
		"Content-Type",
		"application/vnd.apple.mpegurl",
	)
	ctx.Response().WriteHeader(http.StatusOK)
	n, err := io.Copy(ctx.Response(), fileInfo.Reader)
	if err != nil {
		return response.ResponseError(
			ctx,
			response.NewInternalServerError(err),
		)
	}

	fileInfo.Size = n

	return nil
}

func (c *MediaController) GetDemoMediaObject(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaObject").
			Observe(time.Since(t).Seconds())
	}()

	media, err := c.mediaService.GetDemoMedia(
		ctx.Request().Context(),
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return ctx.JSON(http.StatusOK, models.NewMediaObject(media))
}
