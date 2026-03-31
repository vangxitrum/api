package controllers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

type MediaChapterController struct {
	mediaChapterService *services.MediaChapterService
}

func NewMediaChapterController(
	mediaChapterService *services.MediaChapterService,
) *MediaChapterController {
	return &MediaChapterController{
		mediaChapterService: mediaChapterService,
	}
}

type CreateMediaChapterData struct {
	Chapter *models.MediaChapter `json:"media_chapter"`
} //	@name	CreateMediaChapterData

type CreateMediaChapterResponse struct {
	Status string                 `json:"status"`
	Data   CreateMediaChapterData `json:"data"`
} //	@name	CreateMediaChapterResponse

// CreateMediaChapter godoc
//
//	@Summary			Create a media chapter
//	@Description		Create a VTT file to add chapters to your media. Chapters help break the media into sections.
//	@Tags				Media chapter
//	@Id					POST_media-mediaId-chapters-language
//	@Accept				multipart/form-data
//	@Produce			json
//	@Security			BasicAuth
//	@Security			Bearer
//	@Param				id		path		string	true	"Media ID"
//	@Param				lan		path		string	true	"Language"
//	@Param				file	formData	file	true	"VTT File"
//	@Success			201		{object}	CreateMediaChapterResponse
//	@Header				201		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				201		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				201		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
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
//	@Router				/media/{id}/chapters/{lan} [post]
//	@x-client-action	"create"
func (c *MediaChapterController) CreateMediaChapter(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CreateMediaChapter").
			Observe(time.Since(t).Seconds())
	}()

	file, err := ctx.FormFile("file")
	if err != nil {
		return response.NewHttpError(
			http.StatusBadRequest,
			err,
			"Invalid file.",
		)
	}

	if file.Size > models.MaxVTTFileSize {
		return response.NewHttpError(
			http.StatusBadRequest,
			nil,
			"File size is too large (max: 50MB).",
		)
	}

	reader, err := file.Open()
	if err != nil {
		return response.NewHttpError(
			http.StatusBadRequest,
			err,
			"Invalid file.",
		)
	}

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.NewHttpError(
			http.StatusBadRequest,
			err,
			"Invalid media's id.",
		)
	}

	if _, ok := models.ValidLanguageMapping[ctx.Param("lan")]; !ok {
		return response.NewHttpError(
			http.StatusBadRequest,
			nil,
			"Invalid language.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	mediaChapter, err := c.mediaChapterService.Create(
		ctx.Request().Context(),
		mediaId,
		ctx.Param("lan"),
		reader,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusCreated, CreateMediaChapterData{
		Chapter: mediaChapter,
	})
}

type GetMediaChaptersRequest struct {
	Offset int `json:"offset" query:"offset" form:"offset"`
	Limit  int `json:"limit"  query:"limit"  form:"limit"`
} //	@name	GetMediaChaptersRequest

type GetMediaChaptersData struct {
	Chapters []*models.MediaChapter `json:"media_chapters"`
	Total    int64                  `json:"total"`
} //	@name	GetMediaChaptersData

type GetMediaChaptersResponse struct {
	Status string               `json:"status"`
	Data   GetMediaChaptersData `json:"data"`
} //	@name	GetMediaChaptersResponse

// GetMediaChapters godoc
//
//	@Summary			Get media chapters
//	@Description		Get a chapter for by media id in a specific language.
//	@Tags				Media chapter
//	@Id					GET_media-mediaId-chapters-language
//
//	@Accept				json
//	@Produce			json
//	@Security			BasicAuth
//	@Security			Bearer
//
//	@Param				id		path		string	true	"Media ID"
//	@Param				offset	query		integer	false	"offset, allowed values greater than or equal to 0. Default(0)"	minimum(0)	default(0)
//	@Param				limit	query		integer	false	"results per page. Allowed values 1-100, default is 25"			minimum(1)	maximum(100)	default(25)
//
//	@Success			200		{object}	GetMediaChaptersResponse
//	@Failure			400		{object}	models.ResponseError
//	@Failure			403		{object}	models.ResponseError
//	@Failure			404		{object}	models.ResponseError
//	@Failure			500		{object}	models.ResponseError
//
//	@Header				200		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				200		{integer}	X-RateLimit-Remaining	"Available requests left for current window"
//	@Header				200		{integer}	X-RateLimit-Retry-After	"Seconds until rate limit window resets"
//	@Header				400		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400		{integer}	X-RateLimit-Remaining	"Available requests left for current window"
//	@Header				400		{integer}	X-RateLimit-Retry-After	"Seconds until rate limit window resets"
//	@Header				403		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				403		{integer}	X-RateLimit-Remaining	"Available requests left for current window"
//	@Header				403		{integer}	X-RateLimit-Retry-After	"Seconds until rate limit window resets"
//	@Header				404		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				404		{integer}	X-RateLimit-Remaining	"Available requests left for current window"
//	@Header				404		{integer}	X-RateLimit-Retry-After	"Seconds until rate limit window resets"
//	@Header				500		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500		{integer}	X-RateLimit-Remaining	"Available requests left for current window"
//	@Header				500		{integer}	X-RateLimit-Retry-After	"Seconds until rate limit window resets"
//
//	@Router				/media/{id}/chapters [get]
//	@x-client-action	"get"
//	@x-group-parameters	true
//	@x-client-paginated	true
func (c *MediaChapterController) GetMediaChapters(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaChapters").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetMediaChaptersRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.NewHttpError(
			http.StatusBadRequest,
			err,
			"Invalid input.",
		)
	}

	if payload.Offset < 0 && payload.Limit < 0 {
		return response.NewHttpError(
			http.StatusBadRequest,
			nil,
			"Invalid offset or limit.",
		)
	}

	if payload.Limit == 0 || payload.Limit > models.PageSizeLimit {
		payload.Limit = models.PageSizeLimit
	}

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.NewHttpError(
			http.StatusBadRequest,
			err,
			"Invalid media's id.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	mediaChapters, total, err := c.mediaChapterService.GetMediaChapters(
		ctx.Request().Context(),
		mediaId,
		payload.Offset,
		payload.Limit,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, GetMediaChaptersData{
		Chapters: mediaChapters,
		Total:    total,
	})
}

// DeleteMediaChapter godoc
//
//	@Summary			Delete a media chapter
//	@Description		Delete a chapter in a specific language by providing the media ID for the media you want to delete the chapter from and the language the chapter is in.
//	@Tags				Media chapter
//	@Id					DELETE_media-mediaId-chapters-language
//
//	@Accept				json
//	@Produce			json
//	@Security			BasicAuth
//	@Security			Bearer
//
//	@Param				id	path		string	true	"Media ID"
//	@Param				lan	path		string	true	"Language"
//
//	@Success			200	{object}	models.ResponseSuccess
//	@Failure			400	{object}	models.ResponseError
//	@Failure			403	{object}	models.ResponseError
//	@Failure			404	{object}	models.ResponseError
//	@Failure			500	{object}	models.ResponseError
//
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
//
//	@Router				/media/{id}/chapters/{lan} [delete]
//	@x-client-action	"delete"
func (c *MediaChapterController) DeleteMediaChapter(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteMediaChapter").
			Observe(time.Since(t).Seconds())
	}()

	mediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.NewHttpError(
			http.StatusBadRequest,
			err,
			"Invalid media's id.",
		)
	}

	if _, ok := models.ValidLanguageMapping[ctx.Param("lan")]; !ok {
		return response.NewHttpError(
			http.StatusBadRequest,
			nil,
			"Invalid language.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	if err := c.mediaChapterService.DeleteMediaChapter(
		ctx.Request().Context(),
		mediaId,
		ctx.Param("lan"),
		authInfo,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, "Delete chapter successfully.")
}
