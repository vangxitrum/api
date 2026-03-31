package controllers

import (
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

type MediaCaptionController struct {
	mediaCaptionService *services.MediaCaptionService
}

func NewMediaCaptionController(
	mediaCaptionService *services.MediaCaptionService,
) *MediaCaptionController {
	return &MediaCaptionController{
		mediaCaptionService: mediaCaptionService,
	}
}

type CreateMediaCaptionData struct {
	Caption *models.MediaCaption `json:"media_caption"`
} //	@name	CreateMediaCaptionData

type CreateMediaCaptionResponse struct {
	Status string                 `json:"status"`
	Data   CreateMediaCaptionData `json:"data"`
} //	@name	CreateMediaCaptionResponse

type RequestCreateCaption struct {
	Description string `form:"description"`
} //	@name	RequestCreateCaption

// CreateMediaCaption godoc
//
//	@Summary			Create a new media caption
//	@Description		Uploads a VTT file and creates a new media caption for the specified media.
//	@Tags				media
//	@Id					POST_media-mediaId-caption-language
//
//	@Accept				multipart/form-data
//	@Produce			json
//	@Security			BasicAuth
//	@Security			Bearer
//
//	@Param				id		path		string					true	"Media ID"
//	@Param				lan		path		string					true	"Language"
//	@Param				body	formData	RequestCreateCaption	false	"Caption description"
//	@Param				file	formData	file					true	"VTT File"
//
//	@Success			201		{object}	CreateMediaCaptionResponse
//	@Failure			400		{object}	models.ResponseError
//	@Failure			403		{object}	models.ResponseError
//	@Failure			404		{object}	models.ResponseError
//	@Failure			500		{object}	models.ResponseError
//
//	@Header				201		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				201		{integer}	X-RateLimit-Remaining	"Available requests left for current window"
//	@Header				201		{integer}	X-RateLimit-Retry-After	"Seconds until rate limit window resets"
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
//	@Router				/media/{id}/captions/{lan} [post]
//	@x-client-action	"createCaption"
func (c *MediaCaptionController) CreateMediaCaption(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CreateMediaCaption").
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

	var payload RequestCreateCaption
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	description := payload.Description
	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	mediaCaption, err := c.mediaCaptionService.CreateMediaCaption(
		ctx.Request().Context(),
		mediaId,
		ctx.Param("lan"),
		description,
		reader,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusCreated, CreateMediaCaptionData{
		Caption: mediaCaption,
	})
}

type GetMediaCaptionsRequest struct {
	Offset int `json:"offset" query:"offset" form:"offset"`
	Limit  int `json:"limit"  query:"limit"  form:"limit"`
} //	@name	GetMediaSubtitlesData

type GetMediaCaptionsData struct {
	Captions []*models.MediaCaption `json:"media_captions"`
	Total    int64                  `json:"total"`
} //	@name	GetMediaCaptionsData

type GetMediaCaptionsResponse struct {
	Status string               `json:"status"`
	Data   GetMediaCaptionsData `json:"data"`
} //	@name	GetMediaCaptionsResponse

// GetMediaCaptions godoc
//
//	@Summary			Get media captions
//	@Description		Retrieves a list of media captions for the specified media.
//	@Tags				media
//	@Id					GET_media-mediaId-caption-language
//	@Accept				json
//	@Produce			json
//	@Security			BasicAuth
//	@Security			Bearer
//	@Param				id		path		string	true	"Media ID"
//	@Param				offset	query		integer	false	"offset, allowed values greater than or equal to 0. Default(0)"	minimum(0)	default(0)
//	@Param				limit	query		integer	false	"results per page. Allowed values 1-100, default is 25"			minimum(1)	maximum(100)	default(25)
//	@Success			200		{object}	GetMediaCaptionsResponse
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
//	@Router				/media/{id}/captions [get]
//	@x-client-action	"getCaptions"
//	@x-group-parameters	true
//	@x-client-paginated	true
func (c *MediaCaptionController) GetMediaCaptions(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMediaCaptions").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetMediaCaptionsRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.NewHttpError(
			http.StatusBadRequest,
			err,
			"Invalid input.",
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

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	mediaCaptions, total, err := c.mediaCaptionService.GetMediaCaptions(
		ctx.Request().Context(),
		mediaId,
		payload.Offset,
		payload.Limit,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, GetMediaCaptionsData{
		Captions: mediaCaptions,
		Total:    total,
	})
}

type SetDefaultCaptionRequest struct {
	IsDefault bool `json:"is_default" form:"is_default"`
} //	@name	SetDefaultSubtitleRequest

func (c *MediaCaptionController) SetDefaultCaption(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("SetDefaultCaption").
			Observe(time.Since(t).Seconds())
	}()

	var payload SetDefaultCaptionRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.NewHttpError(
			http.StatusBadRequest,
			err,
			"Invalid input.",
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
	if err := c.mediaCaptionService.SetMediaDefaultCaption(
		ctx.Request().Context(),
		mediaId,
		ctx.Param("lan"),
		payload.IsDefault,
		authInfo,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, "Set default caption successfully.")
}

// DeleteMediaCaption godoc
//
//	@Summary			Delete a media caption
//	@Description		Delete a caption in a specific language by providing the media ID for the media you want to delete the caption from and the language the caption is in.
//	@Tags				media
//	@Id					DELETE_media-mediaId-captions-language
//
//	@Accept				json
//	@Produce			json
//	@Security			BasicAuth
//	@Security			Bearer
//
//	@Param				id	path		string					true	"Media ID"
//	@Param				lan	path		string					true	"Language"
//
//	@Success			200	{object}	models.ResponseSuccess	"Delete caption successfully."
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
//	@Router				/media/{id}/captions/{lan} [delete]
//	@x-client-action	"deleteCaption"
func (c *MediaCaptionController) DeleteMediaCaption(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteMediaCaption").
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

	langParam := ctx.Param("lan")
	re := regexp.MustCompile(`^.{0,2}`)
	lang := re.FindString(langParam)
	if _, ok := models.ValidLanguageMapping[lang]; !ok {
		return response.NewHttpError(
			http.StatusBadRequest,
			nil,
			"Invalid language.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	if err := c.mediaCaptionService.DeleteMediaCaption(
		ctx.Request().Context(),
		mediaId,
		ctx.Param("lan"),
		authInfo,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, "Delete caption successfully.")
}
