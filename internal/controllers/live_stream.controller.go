package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/validate"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

type LiveStreamController struct {
	liveStreamService          *services.LiveStreamService
	liveStreamMediaService     *services.LiveStreamMediaService
	LiveStreamMulticastService *services.LiveStreamMulticastService
	liveStreamStatisticService *services.LiveStreamStatisticService
	usageService               *services.UsageService
	mediaService               *services.MediaService
}

func NewLiveStreamController(
	liveStreamService *services.LiveStreamService,
	liveStreamMediaService *services.LiveStreamMediaService,
	liveStreamMulticastService *services.LiveStreamMulticastService,
	liveStreamStatisticService *services.LiveStreamStatisticService,
	usageService *services.UsageService,
	mediaService *services.MediaService,
) *LiveStreamController {
	return &LiveStreamController{
		liveStreamService:          liveStreamService,
		liveStreamMediaService:     liveStreamMediaService,
		LiveStreamMulticastService: liveStreamMulticastService,
		liveStreamStatisticService: liveStreamStatisticService,
		usageService:               usageService,
		mediaService:               mediaService,
	}
}

type CreateLiveStreamKeyRequest struct {
	Name string `json:"name"`
	Save bool   `json:"save"`
	Type string `json:"type"`
} //	@name	CreateLiveStreamKeyRequest

type LiveStreamKeyData struct {
	Id        uuid.UUID `json:"id"`
	UserId    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Save      bool      `json:"save"`
	Type      string    `json:"type"`
	StreamKey uuid.UUID `json:"stream_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	RtmpUrl   string    `json:"rtmp_url"`
} //	@name	LiveStreamKeyData

type CreateLiveStreamKeyResponse struct {
	Status string            `json:"status"`
	Data   LiveStreamKeyData `json:"data"`
} //	@name	CreateLiveStreamKeyResponse

// CreateLiveStreamKey godoc
//
//	@Summary			Create live stream key
//	@Description		Create live stream key
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				input	body		CreateLiveStreamKeyRequest	true	"CreateLiveStreamKeyRequest"
//	@Success			200		{object}	CreateLiveStreamKeyResponse	"CreateLiveStreamKeyResponse"
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
//	@Router				/live_streams [post]
//	@x-client-action	"createLiveStreamKey"
func (c *LiveStreamController) CreateLiveStreamKey(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CreateLiveStreamKey").
			Observe(time.Since(t).Seconds())
	}()
	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	var input models.CreateLiveStreamKeyInput

	if err := ctx.Bind(&input); err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			err.Error(),
		)
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Name must be specified.",
		)
	}

	isAnotherType := input.Type != models.VideoType && input.Type != models.AudioType
	if isAnotherType {
		input.Type = models.VideoType
	}

	result, err := c.liveStreamService.CreateLiveStreamKey(
		ctx.Request().Context(),
		authInfo,
		input,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		models.ConvertLiveStreamKeyToResponse(result),
	)
}

type GetLiveStreamKeyData struct {
	Id        uuid.UUID `json:"id"`
	UserId    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Save      bool      `json:"save"`
	Type      string    `json:"type"`
	StreamKey uuid.UUID `json:"stream_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	RtmpUrl   string    `json:"rtmp_url"`
} //	@name	GetLiveStreamKeyData

type GetLiveStreamKeyResponse struct {
	Status string               `json:"status"`
	Data   GetLiveStreamKeyData `json:"data"`
} //	@name	GetLiveStreamKeyResponse

// GetLiveStreamKey godoc
//
//	@Summary			Get live stream key
//	@Description		Get live stream key
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id	path		string						true	"ID"
//	@Success			200	{object}	GetLiveStreamKeyResponse	"LiveStreamKey"
//	@Header				200	{integer}	X-RateLimit-Limit			"The request limit per minute"
//	@Header				200	{integer}	X-RateLimit-Remaining		"The number of available requests left for the current time window"
//	@Header				200	{integer}	X-RateLimit-Retry-After		"The number of seconds left until the current rate limit window resets"
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
//	@Router				/live_streams/{id} [get]
//	@x-client-action	"getLiveStreamKey"
func (c *LiveStreamController) GetLiveStreamKey(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetLiveStreamKey").
			Observe(time.Since(t).Seconds())
	}()

	uid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream key's id is invalid.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	result, err := c.liveStreamService.GetLiveStreamKeyById(
		ctx.Request().Context(),
		uid,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		models.ConvertLiveStreamKeyToResponse(result),
	)
}

type GetLiveStreamKeysListData struct {
	LiveStreamKeys []GetLiveStreamKeyData `json:"live_stream_keys"`
	Total          int                    `json:"total"`
} //	@name	GetLiveStreamKeysListData

type GetLiveStreamKeysListResponse struct {
	Status string                    `json:"status"`
	Data   GetLiveStreamKeysListData `json:"data"`
} //	@name	GetLiveStreamKeysListResponse

// GetLiveStreamKeys godoc
//
//	@Summary			Get live stream key list
//	@Description		Get live stream key list
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				search		query		string							false	"only support search by name"
//	@Param				sort_by		query		string							false	"sort by"							Enums(created_at, name)	default(created_at)
//	@Param				order_by	query		string							false	"allowed: asc, desc. Default: asc"	Enums(asc,desc)			default(asc)
//	@Param				offset		query		integer							false	"offset, allowed values greater than or equal to 0."
//	@Param				limit		query		integer							false	"results per page."
//	@Param				type		query		string							false	"type of media. Enums(audio, video) default(video)."
//	@Success			200			{object}	GetLiveStreamKeysListResponse	"GetLiveStreamKeysResponse"
//	@Header				200			{integer}	X-RateLimit-Limit				"The request limit per minute"
//	@Header				200			{integer}	X-RateLimit-Remaining			"The number of available requests left for the current time window"
//	@Header				200			{integer}	X-RateLimit-Retry-After			"The number of seconds left until the current rate limit window resets"
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
//	@Router				/live_streams [get]
//	@x-group-parameters	true
//	@x-client-paginated	true
//	@x-optional-object	true
//	@x-client-action	"getLiveStreamKeys"
func (c *LiveStreamController) GetLiveStreamKeys(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetLiveStreamKeys").
			Observe(time.Since(t).Seconds())
	}()
	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	var payload models.GetLiveStreamKeysFilter
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			err.Error(),
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

	if payload.SortBy == "" {
		payload.SortBy = models.DefaultMediaSortBy
	}

	if payload.OrderBy == "" {
		payload.OrderBy = models.DefaultOrderBy
	}

	isAnotherType := payload.Type != models.VideoType && payload.Type != models.AudioType
	isEmptyType := payload.Type == ""
	// If is empty return all video and audio by default
	if !isEmptyType && isAnotherType {
		payload.Type = "video"
	}

	if !models.LiveStreamKeysSortByMap[payload.SortBy] {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			fmt.Sprintf(
				"Sorting by '%s' is not allowed. Allowed values: created_at, name.",
				payload.SortBy,
			),
		)
	}

	if !models.OrderMap[payload.OrderBy] {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Unknown sorting order. Please use \"asc\" or \"desc\".",
		)
	}

	if payload.Limit > int(models.MaxPageLimit) {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Limit only allowed values 1-100.",
		)
	}

	if payload.Limit == 0 {
		payload.Limit = 25
	}

	result, total, err := c.liveStreamService.GetLiveStreamKeys(
		ctx.Request().Context(),
		authInfo.User.Id,
		payload,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}
	liveStreamKeyWithRtmp := models.ConvertListLiveStreamKeyToResponse(result)
	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		models.GetLiveStreamKeysResponse{
			LiveStreamKeys: liveStreamKeyWithRtmp,
			Total:          total,
		},
	)
}

type UpdateLiveStreamKeyRequest struct {
	Name string `json:"name"`
	Save bool   `json:"save"`
	Type bool   `json:"type"`
} //	@name	UpdateLiveStreamKeyRequest

type UpdateLiveStreamKeyData struct {
	Id        uuid.UUID `json:"id"`
	UserId    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Save      bool      `json:"save"`
	Type      string    `json:"type"`
	StreamKey uuid.UUID `json:"stream_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	RtmpUrl   string    `json:"rtmp_url"`
} //	@name	UpdateLiveStreamKeyData

type UpdateLiveStreamKeyResponse struct {
	Status string                  `json:"status"`
	Data   UpdateLiveStreamKeyData `json:"data"`
} //	@name	UpdateLiveStreamKeyResponse

// UpdateLiveStreamKey godoc
//
//	@Summary			Update live stream key
//	@Description		Update a live stream key by ID
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id		path		string						true	"Live stream key ID"
//	@Param				input	body		UpdateLiveStreamKeyRequest	true	"UpdateLiveStreamKeyRequest"
//	@Success			200		{object}	UpdateLiveStreamKeyResponse	"Updated LiveStreamKey"
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
//	@Router				/live_streams/{id} [put]
//	@x-client-action	"updateLiveStreamKey"
func (c *LiveStreamController) UpdateLiveStreamKey(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UpdateLiveStreamKey").
			Observe(time.Since(t).Seconds())
	}()

	uid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream key's id is invalid.",
		)
	}

	var input models.UpdateLiveStreamKeyInput
	if err := ctx.Bind(&input); err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			err.Error(),
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	updatedKey, err := c.liveStreamService.UpdateLiveStreamKey(
		ctx.Request().Context(),
		authInfo.User.Id,
		uid,
		input,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		models.ConvertLiveStreamKeyToResponse(updatedKey),
	)
}

// DeleteLiveStreamKey godoc
//
//	@Summary			Delete live stream key
//	@Description		Delete a live stream key by ID
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id	path		string	true	"Live stream key ID"
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
//	@Router				/live_streams/{id} [delete]
//	@x-client-action	"deleteLiveStreamKey"
func (c *LiveStreamController) DeleteLiveStreamKey(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteLiveStreamKey").
			Observe(time.Since(t).Seconds())
	}()

	uid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream key's id is invalid.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	if err := c.liveStreamService.DeleteLiveStreamKey(
		ctx.Request().Context(),
		uid,
		authInfo.User.Id,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Delete live stream key success.",
	)
}

type GetLiveStreamMediaResponse struct {
	Status string                          `json:"status"`
	Data   *models.LiveStreamMediaResponse `json:"data"`
} //	@name	GetLiveStreamMediaResponse

// GetLiveStreamMedia godoc
//
//	@Summary			Get live stream media
//	@Description		Get a specific live stream media by ID
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id	path		string						true	"Live stream media ID"
//	@Success			200	{object}	GetLiveStreamMediaResponse	"LiveStreamMedia"
//	@Header				200	{integer}	X-RateLimit-Limit			"The request limit per minute"
//	@Header				200	{integer}	X-RateLimit-Remaining		"The number of available requests left for the current time window"
//	@Header				200	{integer}	X-RateLimit-Retry-After		"The number of seconds left until the current rate limit window resets"
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
//	@Router				/live_streams/{id}/media [get]
//	@x-client-action	"getLiveStreamMedia"
func (c *LiveStreamController) GetLiveStreamMedia(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetLiveStreamMedia").
			Observe(time.Since(t).Seconds())
	}()

	uid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream media's id is invalid.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	result, err := c.liveStreamMediaService.GetLiveStreamMediaByIdAndUserId(
		ctx.Request().Context(),
		authInfo.User.Id,
		uid,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		models.ConvertLiveStreamMediaToResponse(result),
	)
}

type GetLiveStreamMediaPublicResponse struct {
	Status string                          `json:"status"`
	Data   *models.LiveStreamMediaResponse `json:"data"`
} //	@name	GetLiveStreamMediaPublicResponse

// GetLiveStreamMediaPublic godoc
//
//	@Summary			Get live stream media public
//	@Description		Get live stream media public for a specific live stream key
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Param				id	path		string								true	"Live stream key ID"
//	@Success			200	{object}	GetLiveStreamMediaPublicResponse	"LiveStreamMedia"
//	@Header				200	{integer}	X-RateLimit-Limit					"The request limit per minute"
//	@Header				200	{integer}	X-RateLimit-Remaining				"The number of available requests left for the current time window"
//	@Header				200	{integer}	X-RateLimit-Retry-After				"The number of seconds left until the current rate limit window resets"
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
//	@Router				/live_streams/player/{id}/media [get]
//	@x-client-action	"getLiveStreamPlayerInfo"
func (c *LiveStreamController) GetLiveStreamMediaPublic(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetLiveStreamMediaPublic").
			Observe(time.Since(t).Seconds())
	}()

	uid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream media's id is invalid.",
		)
	}

	result, err := c.liveStreamMediaService.GetLiveStreamMediaById(
		ctx.Request().Context(),
		uid,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	result.LiveStreamKeyId = uuid.Nil
	result.UserId = uuid.Nil

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		models.ConvertLiveStreamMediaToResponse(result),
	)
}

type LiveStreamMediaData struct {
	Id              uuid.UUID           `json:"id"`
	LiveStreamKeyId uuid.UUID           `json:"live_stream_key_id"`
	UserId          uuid.UUID           `json:"user_id"`
	Title           string              `json:"title"`
	Duration        int64               `json:"duration"`
	Assets          LiveStreamAssets    `json:"assets"`
	Save            bool                `json:"save"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	Status          string              `json:"status"`
	Media           *models.MediaObject `json:"media"`
	Qualities       []string            `json:"qualities"`
} //	@name	LiveStreamMediaData

type LiveStreamAssets struct {
	HlsUrl       string `json:"hls_url"`
	IFrame       string `json:"iframe"`
	PlayerUrl    string `json:"player_url"`
	ThumbnailUrl string `json:"thumbnail_url"`
} //	@name	LiveStreamAssets

type CreateStreamingRequest struct {
	Title string `json:"title"`
	Save  bool   `json:"save"`
	// Qualities of the media (default: 1080p, 720p,  360p, allow:2160p, 1440p, 1080p, 720p,  360p, 240p, 144p)
	Qualities []*models.QualityConfig `json:"qualities" form:"qualities"`
} //	@name	CreateStreamingRequest

type CreateStreamingResponse struct {
	Status string               `json:"status"`
	Data   *LiveStreamMediaData `json:"data"`
} //	@name	CreateStreamingResponse

// CreateStreaming godoc
//
//	@Summary			Create a new live stream media
//	@Description		Creates a new live stream media with the provided details
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id		path		string					true	"Live stream key ID"
//	@Param				input	body		CreateStreamingRequest	true	"CreateStreamingRequest"
//	@Success			200		{object}	CreateStreamingResponse	"Created LiveStreamMedia"
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
//	@Router				/live_streams/{id}/streamings [post]
//	@x-client-action	"createStreaming"
func (c *LiveStreamController) CreateStreaming(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CreateStreaming").
			Observe(time.Since(t).Seconds())
	}()

	streamKeyId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream key's id is invalid.",
		)
	}

	var payload models.CreateStreamingRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseError(ctx, err)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	if err := c.liveStreamMediaService.EnsureEnoughBalanceIfSave(ctx.Request().Context(), authInfo.User.Id, payload.Save); err != nil {
		return response.ResponseError(ctx, err)
	}

	if c.liveStreamMediaService.CheckLiveStreamIsExist(ctx.Request().Context(), streamKeyId, authInfo.User.Id) {
		return response.NewBadRequestError("Live stream already created")
	}

	lsKey, err := c.liveStreamService.GetLiveStreamKeyById(ctx.Request().Context(), streamKeyId, authInfo)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	qualities, err := normalizeAndValidateQualities(ctx, payload.Qualities, lsKey.Type)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	payload.Qualities = qualities
	payload.Title = strings.TrimSpace(payload.Title)
	if payload.Title == "" {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Title is required.",
		)
	}

	if len(payload.Title) > models.TitleMaxLen {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Title is too long.",
		)
	}

	newMedia, err := models.NewMedia(
		authInfo.User.Id,
		lsKey.Type,
		payload.Title,
		"",
		[]models.Metadata{},
		payload.Qualities,
		models.DefaultSegmentDuration,
		[]string{},
		true,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	if _, err := c.mediaService.CreateMediaObjectLiveStream(
		ctx.Request().Context(),
		newMedia,
		streamKeyId,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	lsMedia, err := c.liveStreamMediaService.CreateLiveStreamMedia(
		ctx.Request().Context(),
		newMedia,
		streamKeyId,
		authInfo.User.Id,
		models.LiveStreamStatusCreated,
		models.CreateStreamingRequest{
			Title:     payload.Title,
			Save:      payload.Save,
			Qualities: payload.Qualities,
		},
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		models.ConvertLiveStreamMediaToResponse(lsMedia),
	)
}

type GetStreamingsRequest struct {
	Search  string `json:"search"   query:"search"`
	SortBy  string `json:"sort_by"  query:"sort_by"  form:"sort_by"`
	OrderBy string `json:"order_by" query:"order_by" form:"order_by"`
	Offset  int    `json:"offset"   query:"offset"   form:"offset"`
	Limit   int    `json:"limit"    query:"limit"    form:"limit"`
} //	@name	GetSteamingRequest

type GetStreamingsResponse struct {
	Status string                           `json:"status"`
	Data   *models.LiveStreamMediasResponse `json:"data"`
} //	@name	GetStreamingsResponse

// GetStreamings godoc
//
//	@Summary			Get live stream media streamings
//	@Description		Get live stream media streamings for a specific live stream key
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id		path		string					true	"Live stream key ID"
//	@Param				search	query		string					false	"Search"
//	@Success			200		{object}	GetStreamingsResponse	"LiveStreamMediasResponse"
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
//	@Router				/live_streams/{id}/streamings [get]
//	@x-client-action	"getStreamings"
func (c *LiveStreamController) GetStreamings(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetStreamings").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetStreamingsRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseError(ctx, err)
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

	if payload.Search != "" {
		payload.Search = strings.TrimSpace(payload.Search)
	}

	if _, ok := models.SortByMap[payload.SortBy]; !ok {
		payload.SortBy = models.DefaultMediaSortBy
	}

	if _, ok := models.OrderMap[payload.OrderBy]; !ok {
		payload.OrderBy = models.DefaultOrderBy
	}

	liveStreamKeyId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream media's id is invalid.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	liveStreamMedias, total, err := c.liveStreamMediaService.GetLiveStreamMediaStreamings(
		ctx.Request().Context(),
		liveStreamKeyId,
		models.GetStreamingsFilter{
			Search:  payload.Search,
			SortBy:  payload.SortBy,
			OrderBy: payload.OrderBy,
			Offset:  payload.Offset,
			Limit:   payload.Limit,
		},
		authInfo.User.Id,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		models.ConvertLiveStreamMediasToResponse(liveStreamMedias, total),
	)
}

type GetStreamingResponse struct {
	Status string                          `json:"status"`
	Data   *models.LiveStreamMediaResponse `json:"data"`
} //	@name	GetStreamingResponse

// GetStreaming godoc
//
//	@Summary			Get live stream media streaming
//	@Description		Get live stream media streaming for a specific live stream key
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id			path		string					true	"Live stream key ID"
//	@Param				stream_id	path		string					true	"Stream ID"
//	@Success			200			{object}	GetStreamingResponse	"LiveStreamMedia"
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
//	@Router				/live_streams/{id}/streamings/{stream_id} [get]
//	@x-client-action	"getStreaming"
func (c *LiveStreamController) GetStreaming(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetStreaming").
			Observe(time.Since(t).Seconds())
	}()

	uid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream media's id is invalid.",
		)
	}

	streamId, err := uuid.Parse(ctx.Param("stream_id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Stream's id is invalid.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	liveStreamMedia, err := c.liveStreamMediaService.GetLiveStreamMediaStreaming(
		ctx.Request().Context(),
		authInfo.User.Id,
		uid,
		streamId,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		models.ConvertLiveStreamMediaToResponse(liveStreamMedia),
	)
}

type GetLiveStreamMediasRequest struct {
	LiveStreamKeyId uuid.UUID `json:"live_stream_key_id"`
	Search          string    `json:"search"             form:"search"`
	SortBy          string    `json:"sort_by"            form:"sort_by"`
	OrderBy         string    `json:"order_by"           form:"order_by"`
	Offset          int       `json:"offset"             form:"offset"`
	Limit           int       `json:"limit"              form:"limit"`
	Status          string    `json:"status"             form:"status"`
	MediaStatus     string    `json:"media_status"       form:"media_status"`
} //	@name	GetLiveStreamMediasRequest

type GetLiveStreamMediasResponse struct {
	Status string                           `json:"status"`
	Data   *models.LiveStreamMediasResponse `json:"data"`
} //	@name	GetLiveStreamMediasResponse

// GetLiveStreamMedias godoc
//
//	@Summary			Get live stream media
//	@Description		Get live stream media for a specific live stream key
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id		path		string						true	"Live stream key ID"
//	@Param				data	body		GetLiveStreamMediasRequest	true	"data"
//	@Success			200		{object}	GetLiveStreamMediasResponse	"ok"
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
//	@Router				/live_streams/{id}/media [post]
//	@x-client-action	"getLiveStreamMedias"
func (c *LiveStreamController) GetLiveStreamMedias(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetLiveStreamMedias").
			Observe(time.Since(t).Seconds())
	}()

	var payload models.GetLiveStreamMediasFilter
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseError(ctx, err)
	}

	uid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream media's id is invalid.",
		)
	}

	payload.LiveStreamKeyId = uid

	if !models.LiveStreamMediasSortByMap[payload.SortBy] {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			fmt.Sprintf(
				"Sorting by '%s' is not allowed. Allowed values: created_at, title.",
				payload.SortBy,
			),
		)
	}

	if !models.OrderMap[payload.OrderBy] {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Unknown sorting order. Please use \"asc\" or \"desc\".",
		)
	}
	if payload.Status != "" {
		if !models.ValidLiveStreamStatus[payload.Status] {
			return response.ResponseFailMessage(
				ctx,
				http.StatusBadRequest,
				"Invalid status. Allow status: new, done, fail, deleted, transcoding",
			)
		}
	}

	if payload.MediaStatus != "" {
		if !slices.Contains(models.ValidMediaStatus, payload.MediaStatus) {
			return response.ResponseError(
				ctx,
				response.NewHttpError(
					http.StatusBadRequest,
					nil,
					"Invalid status. Allow status: "+strings.Join(
						models.ValidMediaStatus,
						", ",
					),
				),
			)
		}
	}

	if payload.Limit > int(models.MaxPageLimit) {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Limit only allowed values 1-100.",
		)
	}

	if payload.Limit == 0 {
		payload.Limit = 25
	}

	if payload.SortBy == "" {
		payload.SortBy = "created_at"
	}

	if payload.OrderBy == "" {
		payload.OrderBy = "asc"
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	result, total, err := c.liveStreamMediaService.GetLiveStreamMedias(
		ctx.Request().Context(),
		authInfo.User.Id,
		payload,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		models.ConvertLiveStreamMediasToResponse(result, total),
	)
}

type UpdateLiveStreamMediaRequest struct {
	StreamID uuid.UUID `json:"stream_id"`
	Title    string    `json:"title"`
	Save     bool      `json:"save"`
	// Qualities of the media (default: 1080p, 720p,  360p, allow:2160p, 1440p, 1080p, 720p,  360p, 240p, 144p)
	Qualities []*models.QualityConfig `json:"qualities" form:"qualities"`
} //	@name	UpdateLiveStreamMediaRequest

// UpdateLiveStreamMedia godoc
//
//	@Summary			Update live stream media
//	@Description		Update live stream media. You can only update while live streaming.
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id		path		string							true	"Live stream key ID"
//	@Param				data	body		UpdateLiveStreamMediaRequest	true	"data"
//	@Success			200		{object}	models.ResponseSuccess			"LiveStreamMedia"
//	@Header				200		{integer}	X-RateLimit-Limit				"The request limit per minute"
//	@Header				200		{integer}	X-RateLimit-Remaining			"The number of available requests left for the current time window"
//	@Header				200		{integer}	X-RateLimit-Retry-After			"The number of seconds left until the current rate limit window resets"
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
//	@Router				/live_streams/{id}/streamings [put]
//	@x-client-action	"updateLiveStreamMedia"
func (c *LiveStreamController) UpdateLiveStreamMedia(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UpdateLiveStreamMedia").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	var payload models.UpdateLiveStreamMediaInput
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseError(ctx, err)
	}
	if err := c.liveStreamMediaService.EnsureEnoughBalanceIfSave(ctx.Request().Context(), authInfo.User.Id, payload.Save); err != nil {
		return response.ResponseError(ctx, err)
	}

	streamKeyId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream media's id is invalid.",
		)
	}

	if payload.Title == "" {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Title must be specified.",
		)
	}

	payload.Title = strings.TrimSpace(payload.Title)
	if len(payload.Title) > models.TitleMaxLen {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			fmt.Sprintf(
				"Title length must be less than %d characters.",
				models.TitleMaxLen,
			),
		)
	}

	lsKey, err := c.liveStreamService.GetLiveStreamKeyById(ctx.Request().Context(), streamKeyId, authInfo)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	livestreamMedia, err := c.liveStreamMediaService.GetLiveStreamMediaStreaming(ctx.Request().Context(), authInfo.User.Id, lsKey.Id, payload.StreamId)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	qualities, err := normalizeAndValidateQualities(ctx, payload.Qualities, livestreamMedia.Type)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	if livestreamMedia.Status != models.LiveStreamStatusStreaming && livestreamMedia.Status != models.LiveStreamStatusCreated {
		return response.NewBadRequestError("This livestream was ended.")
	}

	if err := c.mediaService.UpdateMediaQualities(ctx.Request().Context(), livestreamMedia.MediaId, qualities, authInfo); err != nil {
		return response.ResponseError(ctx, err)
	}

	if err := c.liveStreamMediaService.UpdateLiveStreamMedia(
		ctx.Request().Context(),
		payload.StreamId,
		authInfo.User.Id,
		&payload,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Update live stream media success.",
	)
}

func (c *LiveStreamController) DeleteStreaming(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteStreaming").
			Observe(time.Since(t).Seconds())
	}()

	liveStreamKeyId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream media's id is invalid.",
		)
	}

	liveStreamMediaId, err := uuid.Parse(ctx.Param("stream_id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Streaming's id is invalid.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	liveStreamMedia, err := c.liveStreamMediaService.GetUserLiveStreamMediaByIdAndLiveStreamKeyId(
		ctx.Request().Context(),
		authInfo.User.Id,
		liveStreamKeyId,
		liveStreamMediaId,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	existMedia, err := c.mediaService.GetMediaById(
		ctx.Request().Context(),
		liveStreamMedia.MediaId,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	if existMedia.Status != models.DeletedStatus {
		if err := c.mediaService.DeleteMedia(
			ctx.Request().Context(),
			liveStreamMedia.MediaId,
			authInfo,
		); err != nil {
			return response.ResponseError(ctx, err)
		}
	}

	if err := c.liveStreamMediaService.DeleteLiveStreamMedia(
		ctx.Request().Context(),
		authInfo.User.Id,
		liveStreamMedia,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Delete streaming success.",
	)
}

func (c *LiveStreamController) GetConnectLiveStreamWebhook(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetConnectLiveStreamWebhook").
			Observe(time.Since(t).Seconds())
	}()

	connType := ctx.QueryParam("conn_type")
	connID := ctx.QueryParam("conn_id")

	if connType == "" || connID == "" {
		return response.NewBadRequestError(
			"Missing required parameters is " + connType + " and " + connID + ".",
		)
	}

	if err := c.liveStreamMediaService.GetConnectLiveStreamMediaIdByConnId(ctx.Request().Context(), connID); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Get live stream webhook success.",
	)
}

func (c *LiveStreamController) GetDisconnectLiveStreamWebhook(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetDisconnectLiveStreamWebhook").
			Observe(time.Since(t).Seconds())
	}()

	connType := ctx.QueryParam("conn_type")
	connID := ctx.QueryParam("conn_id")

	if connType == "" || connID == "" {
		return response.NewBadRequestError("Missing required parameters.")
	}

	if connType != "rtmpConn" {
		return response.NewBadRequestError(
			"Invalid connection type: ." + connType,
		)
	}

	if err := c.liveStreamMediaService.HandleDisconnectLiveStreamMediaIdByConnId(
		ctx.Request().Context(),
		connID,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Get disconnect live stream webhook success.",
	)
}

// #region Multicast

type GetLiveStreamMulticastsResponse struct {
	Status string                      `json:"status"`
	Data   *models.LiveStreamMulticast `json:"data"`
} //	@name	GetLiveStreamMulticastResponse

// AddLiveStreamMulticast godoc
//
//	@Summary			Add live stream multicast
//	@Description		Add live stream multicast
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				stream_key	path		string							true	"Live stream key. Use uuid"
//	@Param				data		body		UpsertLiveStreamMulticastInput	true	"data"
//	@Success			200			{object}	GetLiveStreamMulticastResponse	"ok"
//	@Header				200			{integer}	X-RateLimit-Limit				"The request limit per minute"
//	@Header				200			{integer}	X-RateLimit-Remaining			"The number of available requests left for the current time window"
//	@Header				200			{integer}	X-RateLimit-Retry-After			"The number of seconds left until the current rate limit window resets"
//	@Failure			400			{object}	models.ResponseError
//	@Header				400			{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400			{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400			{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500			{object}	models.ResponseError
//	@Header				500			{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500			{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500			{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router				/live_streams/multicast/{stream_key} [post]
//	@x-client-action	"addLiveStreamMulticasts"
func (c *LiveStreamController) AddLiveStreamMulticast(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("Adding Livestream Multicast").
			Observe(time.Since(t).Seconds())
	}()

	var input models.UpsertLiveStreamMulticastInput
	streamKey, err := uuid.Parse(ctx.Param("stream_key"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream key's id is invalid.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	if err := ctx.Bind(&input); err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			err.Error(),
		)
	}

	if len(input.MulticastUrls) == 0 || streamKey == uuid.Nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"StreamKey and MulticastUrls must be specified.",
		)
	}

	if len(input.MulticastUrls) > 2 {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Only 2 stream can forward.")
	}

	// Check valid multicast URLs
	// All URLs must be valid and supported protocols to ensure proper functionality.
	// This validation prevents invalid or unsupported URLs from being processed.
	for _, url := range input.MulticastUrls {
		if !validate.IsValidProtocolsSupported(url) {
			return response.ResponseFailMessage(
				ctx,
				http.StatusBadRequest,
				"Multicast url is invalid.",
			)
		}
	}

	result, err := c.LiveStreamMulticastService.UpsertLiveStreamMulticastUrls(
		ctx.Request().Context(),
		streamKey,
		input.MulticastUrls,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		result,
	)
}

// GetLiveStreamMulticastByStreamKey godoc
//
//	@Summary			Get live stream multicast by stream key
//	@Description		Get live stream multicast by stream key
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				stream_key	path		string							true	"Live stream key. UUID string format"
//	@Success			200			{object}	GetLiveStreamMulticastResponse	"ok"
//	@Header				200			{integer}	X-RateLimit-Limit				"The request limit per minute"
//	@Header				200			{integer}	X-RateLimit-Remaining			"The number of available requests left for the current time window"
//	@Header				200			{integer}	X-RateLimit-Retry-After			"The number of seconds left until the current rate limit window resets"
//	@Failure			400			{object}	models.ResponseError
//	@Header				400			{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400			{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400			{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500			{object}	models.ResponseError
//	@Header				500			{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500			{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500			{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router				/live_streams/multicast/{stream_key} [get]
//	@x-client-action	"getLiveStreamMulticastByStreamKey"
func (c *LiveStreamController) GetLiveStreamMulticastByStreamKey(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetLivestreamMulticast").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	streamKey, err := uuid.Parse(ctx.Param("stream_key"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream key's id is invalid.",
		)
	}

	result, err := c.LiveStreamMulticastService.GetLiveStreamMulticastByStreamKey(
		ctx.Request().Context(),
		streamKey,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}
	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		result,
	)
}

// DeleteLiveStreamMulticast godoc
//
//	@Summary			Delete live stream multicast
//	@Description		Delete live stream multicast
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				stream_key	path		string					true	"Live stream key. UUID string format"
//	@Success			200			{object}	models.ResponseSuccess	"LiveStreamMulticast"
//	@Header				200			{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				200			{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				200			{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			400			{object}	models.ResponseError
//	@Header				400			{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400			{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400			{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500			{object}	models.ResponseError
//	@Header				500			{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500			{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500			{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router				/live_streams/multicast/{stream_key} [delete]
//	@x-client-action	"deleteLiveStreamMulticast"
func (c *LiveStreamController) DeleteLiveStreamMulticast(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteLivestreamMulticast").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	streamKey, err := uuid.Parse(ctx.Param("stream_key"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream key's id is invalid.",
		)
	}

	err = c.LiveStreamMulticastService.DeleteLiveStreamMulticast(
		ctx.Request().Context(),
		streamKey,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		nil,
	)
}

// #endregion

// #region LiveStreamStatistic
type GetLiveStreamStatisticResponse struct {
	Status string                          `json:"status"`
	Data   *models.LiveStreamStatisticResp `json:"data"`
} //	@name	GetLiveStreamStatisticResponse

// GetLiveStreamStatisticByStreamMediaId godoc
//
//	@Summary			Get live stream statistic by stream media id
//	@Description		Get live stream statistic by stream media id
//	@Tags				live_stream
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				stream_media_id	path		string							true	"Live stream media ID"
//	@Success			200				{object}	GetLiveStreamStatisticResponse	"ok"
//	@Header				200				{integer}	X-RateLimit-Limit				"The request limit per minute"
//	@Header				200				{integer}	X-RateLimit-Remaining			"The number of available requests left for the current time window"
//	@Header				200				{integer}	X-RateLimit-Retry-After			"The number of seconds left until the current rate limit window resets"
//	@Failure			400				{object}	models.ResponseError
//	@Header				400				{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400				{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400				{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500				{object}	models.ResponseError
//	@Header				500				{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500				{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500				{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router				/live_streams/statistic/{stream_media_id} [get]
//	@x-client-action	"getLiveStreamStatisticByStreamMediaId"
func (c *LiveStreamController) GetLiveStreamStatisticByStreamMediaId(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetLivestreamStatistic").
			Observe(time.Since(t).Seconds())
	}()

	streamMediaId, err := uuid.Parse(ctx.Param("stream_media_id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Live stream key's id is invalid.",
		)
	}

	result, err := c.liveStreamStatisticService.GetStatisticByStreamMediaId(
		ctx.Request().Context(),
		streamMediaId,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}
	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		result,
	)
}

// endregion

// UploadLiveStreamThumbnail godoc
//
//	@Summary			Upload live stream media thumbnail
//	@Tags				live_stream
//	@Id					POST-live-stream-media-thumbnail
//	@Accept				multipart/form-data
//	@Security			BasicAuth
//	@Security			Bearer
//	@Param				id		path		string	true	"live stream media's id"
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
//	@Router				/live_streams/{id}/thumbnail [post]
//	@x-client-action	"uploadThumbnail"
func (c *LiveStreamController) UploadLiveStreamThumbnail(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UploadLiveStreamThumbnail").
			Observe(time.Since(t).Seconds())
	}()

	livestreamMediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(ctx, response.NewHttpError(http.StatusBadRequest, err, "Invalid livestream media id."))
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return response.ResponseError(ctx, response.NewHttpError(http.StatusBadRequest, err, "Invalid file."))
	}

	if fileHeader.Size > models.MaxThumbnailSize {
		return response.ResponseError(ctx, response.NewHttpError(http.StatusBadRequest, err, "File size is too large. (max size: 8MB)"))
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		return response.ResponseError(ctx, response.NewHttpError(http.StatusBadRequest, err, "Invalid file."))
	}

	file, ok := form.File["file"]
	if !ok || len(file) == 0 {
		return response.ResponseError(ctx, response.NewHttpError(http.StatusBadRequest, err, "Invalid file."))
	}

	src, err := file[0].Open()
	if err != nil {
		return response.ResponseError(ctx, response.NewHttpError(http.StatusBadRequest, err, "Invalid file."))
	}
	defer src.Close()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	mediaLs, err := c.liveStreamMediaService.GetLiveStreamMediaById(ctx.Request().Context(), livestreamMediaId)
	if err != nil {
		return response.NewNotFoundError(err)
	}

	if err := c.mediaService.UploadThumbnail(ctx.Request().Context(), mediaLs.MediaId, src, authInfo); err != nil {
		return response.ResponseError(ctx, response.NewHttpError(http.StatusInternalServerError, err, "Failed to upload thumbnail."))
	}

	return response.ResponseSuccess(ctx, http.StatusCreated, "Uploaded thumbnail successfully.")
}

// DeleteLiveStreamThumbnail godoc
//
//	@Summary			Delete live stream media thumbnail
//	@Tags				live_stream
//	@Id					DELETE-live-stream-media-thumbnail
//	@Security			BasicAuth
//	@Security			Bearer
//	@Param				id	path		string	true	"live stream media's id"
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
//	@Router				/live_streams/{id}/thumbnail [delete]
//	@x-client-action	"deleteThumbnail"
func (c *LiveStreamController) DeleteLiveStreamThumbnail(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteLiveStreamThumbnail").
			Observe(time.Since(t).Seconds())
	}()

	livestreamMediaId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseError(ctx, response.NewHttpError(http.StatusBadRequest, err, "Invalid livestream media id."))
	}

	authInfo, ok := ctx.Get("authInfo").(models.AuthenticationInfo)
	if !ok {
		return response.ResponseFailMessage(ctx, http.StatusUnauthorized, "Unauthorized.")
	}

	mediaLs, err := c.liveStreamMediaService.GetLiveStreamMediaById(ctx.Request().Context(), livestreamMediaId)
	if err != nil {
		return response.NewNotFoundError(err)
	}

	if err := c.mediaService.DeleteMediaThumbnail(ctx.Request().Context(), mediaLs.MediaId, authInfo); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		nil,
	)
}

func normalizeAndValidateQualities(
	ctx echo.Context,
	qualities []*models.QualityConfig,
	typeStream string) ([]*models.QualityConfig, error) {
	if len(qualities) == 0 {
		switch typeStream {
		case models.AudioType:
			qualities = models.DefaultAudioConfig
		case models.VideoType:
			qualities = models.DefaultVideoConfig
		}

		return qualities, nil
	}

	if typeStream == models.AudioType {
		return processAudioTypeQualities(qualities)
	}

	if typeStream == models.VideoType {
		return processVideoTypeQualities(qualities)
	}

	return nil, errors.New("Invalid quality type.")
}

func processAudioTypeQualities(qualities []*models.QualityConfig) ([]*models.QualityConfig, error) {
	for _, quality := range qualities {
		if message, ok := quality.IsValid(models.AudioType); !ok {
			return nil, errors.New(message)
		}

		if quality.AudioConfig == nil {
			defaultQualityConfig, ok := models.DefaultConfigMapping[quality.Resolution]
			if !ok {
				return nil, errors.New("Invalid audio resolution.")
			}

			quality.AudioConfig = defaultQualityConfig.AudioConfig
		}
	}

	return qualities, nil
}

func processVideoTypeQualities(qualities []*models.QualityConfig) ([]*models.QualityConfig, error) {
	var (
		newQualities              []*models.QualityConfig
		haveVideo                 bool
		shouldAddDefaultHlsAudio  bool
		shouldAddDefaultDashAudio bool
	)

	for _, quality := range qualities {

		var (
			isHaveConfigVideo         = quality.VideoConfig != nil
			isVideoDontHaveResolution = isHaveConfigVideo && quality.Resolution == ""
			isEmptyConfigMedia        = quality.VideoConfig == nil && quality.AudioConfig == nil
		)

		if isVideoDontHaveResolution {
			return nil, response.NewHttpError(http.StatusBadRequest, errors.New("Quality's resolution is required."), "Quality's resolution is required.")
		}

		if message, ok := quality.IsValid(models.VideoType); !ok {
			return nil, response.NewHttpError(http.StatusBadRequest, errors.New(message), message)
		}

		if isEmptyConfigMedia {
			if quality.Type == models.HlsQualityType {
				shouldAddDefaultHlsAudio = true
			}

			if quality.Type == models.DashQualityType {
				shouldAddDefaultDashAudio = true
			}

			if quality.Resolution == "" {
				return nil, response.NewHttpError(http.StatusBadRequest, errors.New("Quality's resolution is required."), "Quality's resolution is required.")
			}

			defaultQualityConfig := models.DefaultConfigMapping[quality.Resolution]
			quality.VideoConfig = defaultQualityConfig.VideoConfig
			isHaveConfigVideo = true
		} else if isHaveConfigVideo {
			defaultQualityConfig, ok := models.DefaultConfigMapping[quality.Resolution]
			if !ok {
				return nil, response.NewHttpError(http.StatusBadRequest, errors.New("Invalid qualities."), "Invalid qualities.")
			}

			quality.VideoConfig.Width = defaultQualityConfig.VideoConfig.Width
			quality.VideoConfig.Height = defaultQualityConfig.VideoConfig.Height
		}

		if !haveVideo && isHaveConfigVideo {
			haveVideo = true
		}

		newQualities = append(newQualities, quality)
	}

	if shouldAddDefaultHlsAudio {
		newQualities = append(newQualities, defaultHlsAudioQuality())
	}

	if shouldAddDefaultDashAudio {
		newQualities = append(newQualities, defaultDashAudioQuality())
	}

	if !haveVideo {
		return nil, response.NewHttpError(http.StatusBadRequest, errors.New("Media quality is required."), "Media quality is required.")
	}

	return newQualities, nil
}

func defaultHlsAudioQuality() *models.QualityConfig {
	return &models.QualityConfig{
		Resolution:    "default-audio",
		Type:          models.HlsQualityType,
		ContainerType: models.MpegtsContainerType,
		AudioConfig: &models.AudioConfig{
			Codec:      models.AacCodec,
			Bitrate:    128_000,
			Index:      0,
			SampleRate: 44_100,
		},
	}
}

func defaultDashAudioQuality() *models.QualityConfig {
	return &models.QualityConfig{
		Resolution:    "default-audio",
		Type:          models.DashQualityType,
		ContainerType: models.Mp4ContainerType,
		AudioConfig: &models.AudioConfig{
			Codec:      models.AacCodec,
			Bitrate:    128_000,
			Index:      0,
			SampleRate: 44_100,
		},
	}
}
