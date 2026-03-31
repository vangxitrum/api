package controllers

import (
	"bytes"
	"image"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	imageHelper "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/image"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	utils "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/payload"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

type WatermarkController struct {
	watermarkService *services.WatermarkService
}

func NewWatermarkController(watermarkService *services.WatermarkService) *WatermarkController {
	return &WatermarkController{
		watermarkService: watermarkService,
	}
}

type GetAllWatermarkRequest struct {
	SortBy  string `json:"sort_by"  form:"sort_by"  query:"sort_by"`
	OrderBy string `json:"order_by" form:"order_by" query:"order_by"`
	Offset  uint64 `json:"offset"   form:"offset"   query:"offset"`
	Limit   uint64 `json:"limit"    form:"limit"    query:"limit"`
}

type GetAllWatermarkData struct {
	Watermarks []*models.Watermark `json:"watermarks"`
	Total      int64               `json:"total"`
} //	@name	GetAllWatermarkData

type GetAllWatermarkResponse struct {
	Status string              `json:"status"`
	Data   GetAllWatermarkData `json:"data"`
} //	@name	GetAllWatermarkResponse

// ListAllWaterMarks godoc
//
//	@Summary			List all watermarks
//	@Description		List all watermarks associated with your workspace.
//	@Tags				watermark
//	@Id					LIST-watermarks
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				sort_by		query		string	false	"sort by"														Enums(created_at, name)	default(created_at)
//	@Param				order_by	query		string	false	"allowed: asc, desc. Default: asc"								Enums(asc,desc)			default(asc)
//	@Param				offset		query		integer	false	"offset, allowed values greater than or equal to 0. Default(0)"	minimum(0)				default(0)
//	@Param				limit		query		integer	false	"results per page. Allowed values 1-100, default is 25"			minimum(1)				maximum(100)	default(25)
//	@Success			200			{object}	GetAllWatermarkResponse
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
//	@Router				/watermarks [get]
//	@x-group-parameters	true
//	@x-client-paginated	true
//	@x-client-action	"list"
func (c *WatermarkController) ListAllWaterMarks(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("ListAllWaterMarks").
			Observe(time.Since(t).Seconds())
	}()
	var payload GetAllWatermarkRequest

	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid request payload.")
	}

	if payload.SortBy != "" && !models.SortByMap[payload.SortBy] {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			" Allowed values: created_at, name.",
		)
	}

	if payload.OrderBy != "" && !models.OrderMap[payload.OrderBy] {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Unknown sorting order. Please use \"asc\" or \"desc\".",
		)
	}

	if payload.Limit > models.MaxPageLimit {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Limit only allowed values 1-100.",
		)
	}
	filterPayload := &utils.FilterPayload{
		Limit:  payload.Limit,
		SortBy: payload.SortBy,
		Order:  payload.OrderBy,
	}

	utils.SetDefaultsFilter(filterPayload, 25, "created_at", "asc")

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	filter := models.GetWatermarkList{
		UserId: authInfo.User.Id,
		SortBy: filterPayload.SortBy,
		Order:  filterPayload.Order,
		Offset: payload.Offset,
		Limit:  filterPayload.Limit,
	}
	result, total, err := c.watermarkService.ListAllWatermarks(
		ctx.Request().Context(),
		filter,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		GetAllWatermarkData{
			Watermarks: result,
			Total:      total,
		},
	)
}

type CreateWatermarkData struct {
	WatermarkId string    `json:"watermark_id"`
	CreatedAt   time.Time `json:"created_at"`
} //	@name	CreateWatermarkData

type CreateWatermarkResponse struct {
	Status string              `json:"status"`
	Data   CreateWatermarkData `json:"data"`
} //	@name	CreateWatermarkResponse

// CreateWaterMark godoc
//
//	@Summary			Create a new watermark
//	@Description		Create a new watermark by uploading a JPG or a PNG image.
//	@Tags				watermark
//	@Id					POST_watermark
//	@Accept				multipart/form-data
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				file	formData	file	true	"Watermark image file"	format(jpg, jpeg, png)
//	@Success			201		{object}	CreateWatermarkResponse
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
//	@Router				/watermarks [post]
//	@x-client-action	"upload"
func (c *WatermarkController) CreateWaterMark(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CreateWaterMark").
			Observe(time.Since(t).Seconds())
	}()
	userId := ctx.Get("authInfo").(models.AuthenticationInfo).User.Id
	file, err := ctx.FormFile("file")
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"File watermark image is required.",
		)
	}
	fullFileName := file.Filename

	src, err := file.Open()
	if err != nil {
		return response.NewInternalServerError(err)
	}
	defer src.Close()

	buff := make([]byte, 512)
	_, err = src.Read(buff)
	if err != nil {
		return response.NewInternalServerError(err)
	}
	fileType := http.DetectContentType(buff)

	if err := imageHelper.CheckFileType(fileType); err != nil {
		return response.NewHttpError(http.StatusBadRequest, err)
	}

	img, _, err := image.DecodeConfig(bytes.NewReader(buff))
	if err != nil {
		return response.NewInternalServerError(err)
	}
	width := img.Width
	height := img.Height

	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return response.NewInternalServerError(err)
	}
	result, err := c.watermarkService.UploadWatermark(
		ctx.Request().Context(),
		userId,
		fullFileName,
		int64(width),
		int64(height),
		src,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusCreated,
		CreateWatermarkData{
			WatermarkId: result.Id.String(),
			CreatedAt:   result.CreatedAt,
		},
	)
}

// DeleteWatermarkById godoc
//
//	@Summary			Delete a watermark by ID
//	@Description		Delete a watermark.
//	@Tags				watermark
//	@Id					DELETE_watermark
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id	path		string	true	"Watermark ID"
//	@Success			200	{object}	models.ResponseSuccess
//	@Success			200	{object}	GetWebhooksListResponse	"Get list webhooks"
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
//	@Router				/watermarks/{id} [delete]
//	@x-client-action	"delete"
func (c *WatermarkController) DeleteWatermarkById(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteWaterMark").
			Observe(time.Since(t).Seconds())
	}()
	watermarkId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Watermark's id is invalid.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	if err := c.watermarkService.DeleteWatermarkById(
		ctx.Request().Context(),
		authInfo.User.Id,
		watermarkId,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Deleted watermark successfully.",
	)
}
