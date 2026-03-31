package controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	utils "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/payload"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/validate"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

type WebhookController struct {
	webhookService *services.WebhookService
}

func NewWebhookController(webhookService *services.WebhookService) *WebhookController {
	return &WebhookController{
		webhookService: webhookService,
	}
}

type CreateWebhookRequest struct {
	Name                  string `body:"name"              json:"name"`
	Url                   string `body:"url"               json:"url"`
	FileReceived          bool   `body:"file_received"     json:"file_received"`
	EventEncodingStarted  bool   `body:"encoding_started"  json:"encoding_started"`
	EventEncodingFinished bool   `body:"encoding_finished" json:"encoding_finished"`
	EventEncodingFailed   bool   `body:"encoding_failed"   json:"encoding_failed"`
	EventPartialFinished  bool   `body:"partial_finished"  json:"partial_finished"`
} //	@name	CreateWebhookRequest

type CreateWebhookData struct {
	Webhook *models.Webhook `json:"webhook"`
} //	@name	CreateWebhookData

type CreateWebhookResponse struct {
	Status string            `json:"status"`
	Data   CreateWebhookData `json:"data"`
} //	@name	CreateWebhookResponse

// CreateWebhook godoc
//
//	@Summary			Create webhook
//	@Description		Webhooks can push notifications to your server, rather than polling w3stream for changes
//	@Tags				webhook
//	@Id					POST-webhooks
//
//	@Accept				json
//	@Accept				x-www-form-urlencoded
//	@Produce			json
//	@Security			BasicAuth
//	@Security			Bearer
//	@Param				request	body		CreateWebhookRequest	true	"Create Webhook input"
//	@Success			201		{object}	CreateWebhookResponse
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
//	@Router				/webhooks [post]
//	@x-client-action	"create"
func (c *WebhookController) CreateWebhook(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CreateWebhook").
			Observe(time.Since(t).Seconds())
	}()

	var payload CreateWebhookRequest

	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid request payload.")
	}

	if !validate.IsValidUrl(payload.Url) {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid URL.")
	}

	if payload.Name != "" && len(payload.Name) > 100 {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Name must be less than 100 characters.",
		)
	}

	if strings.TrimSpace(payload.Name) == "" {
		payload.Name = "default-name-webhook"
	}

	if !payload.FileReceived &&
		!payload.EventEncodingStarted &&
		!payload.EventEncodingFinished &&
		!payload.EventEncodingFailed &&
		!payload.EventPartialFinished {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Event must be selected.")
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	input := models.CreateWebhookInput{
		Name:                  strings.TrimSpace(payload.Name),
		UserId:                authInfo.User.Id,
		Url:                   strings.TrimSpace(payload.Url),
		EventFileReceived:     payload.FileReceived,
		EventEncodingStarted:  payload.EventEncodingStarted,
		EventEncodingFinished: payload.EventEncodingFinished,
		EventEncodingFailed:   payload.EventEncodingFailed,
		EventPartialFinished:  payload.EventPartialFinished,
	}

	result, err := c.webhookService.CreateWebhook(
		ctx.Request().Context(),
		input,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusCreated,
		CreateWebhookData{
			Webhook: result,
		},
	)
}

type GetWebhooksListRequest struct {
	Search                string `json:"search"            form:"search"            query:"search"`
	EventEncodingFinished bool   `json:"encoding_finished" form:"encoding_finished" query:"encoding_finished"`
	EventEncodingStarted  bool   `json:"encoding_started"  form:"encoding_started"  query:"encoding_started"`
	EventFileReceived     bool   `json:"file_received"     form:"file_received"     query:"file_received"`
	EncodingFailed        bool   `json:"encoding_failed"   form:"encoding_failed"   query:"encoding_failed"`
	PartialFinished       bool   `json:"partial_finished"  form:"partial_finished"  query:"partial_finished"`
	Offset                uint64 `json:"offset"            form:"offset"            query:"offset"`
	Limit                 uint64 `json:"limit"             form:"limit"             query:"limit"`
	OrderBy               string `json:"order_by"          form:"order_by"          query:"order_by"`
	SortBy                string `json:"sort_by"           form:"sort_by"           query:"sort_by"`
} //	@name	GetWebhooksListRequest

type GetWebhooksListData struct {
	Webhooks []*models.Webhook `json:"webhooks"`
	Total    int64             `json:"total"`
} //	@name	GetWebhooksListData

type GetWebhooksListResponse struct {
	Status string              `json:"status"`
	Data   GetWebhooksListData `json:"data"`
} //	@name	GetWebhooksListResponse

// GetWebhookList godoc
//
//	@Summary				Get list webhooks
//	@Description			Retrieve a list of all webhooks configured for the current workspace.
//	@Tags					webhook
//	@Accept					json
//	@Id						LIST-webhooks
//	@Accept					x-www-form-urlencoded
//	@Produce				json
//	@Security				BasicAuth
//	@Security				Bearer
//	@Param					search				query		string					false	"only support search by name"
//	@Param					sort_by				query		string					false	"sort by"														Enums(created_at, name)	default(created_at)
//	@Param					order_by			query		string					false	"allowed: asc, desc. Default: asc"								Enums(asc,desc)			default(asc)
//	@Param					offset				query		integer					false	"offset, allowed values greater than or equal to 0. Default(0)"	minimum(0)				default(0)
//	@Param					limit				query		integer					false	"results per page. Allowed values 1-100, default is 25"			minimum(1)				maximum(100)	default(25)
//	@Param					encoding_finished	query		bool					false	"search by event encoding finished"
//	@Param					encoding_started	query		bool					false	"search by event encoding started"
//	@Param					file_received		query		bool					false	"search by event file received"
//	@Success				200					{object}	GetWebhooksListResponse	"Get list webhooks"
//	@Header					200					{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					200					{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					200					{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure				400					{object}	models.ResponseError
//	@Header					400					{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					400					{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					400					{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure				403					{object}	models.ResponseError
//	@Header					403					{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					403					{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					403					{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure				404					{object}	models.ResponseError
//	@Header					404					{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					404					{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					404					{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure				500					{object}	models.ResponseError
//	@Header					500					{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					500					{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					500					{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router					/webhooks [get]
//	@x-client-description	{"default":"This method returns a list of your webhooks (with all their details). \nYou can filter what the webhook list that the API returns using the parameters described below."}
//	@x-group-parameters		true
//	@x-client-paginated		true
//	@x-optional-object		true
//	@x-client-action		"list"
func (c *WebhookController) GetWebhookList(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetWebhookList").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetWebhooksListRequest

	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid request payload.")
	}

	if payload.SortBy != "" && !models.WebhookSortByMap[payload.SortBy] {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Allowed values: created_at, url, name.",
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

	input := models.GetWebhookListFilter{
		UserId:                authInfo.User.Id,
		EventEncodingFinished: payload.EventEncodingFinished,
		EventEncodingStarted:  payload.EventEncodingStarted,
		EventFileReceived:     payload.EventFileReceived,
		EventEncodingFailed:   payload.EncodingFailed,
		EventPartialFinished:  payload.PartialFinished,
		Search:                payload.Search,
		Offset:                payload.Offset,
		Limit:                 filterPayload.Limit,
		SortBy:                filterPayload.SortBy,
		Order:                 filterPayload.Order,
	}

	result, total, err := c.webhookService.GetWebhookList(
		ctx.Request().Context(),
		input,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		GetWebhooksListData{
			Webhooks: result,
			Total:    total,
		},
	)
}

type GetUserWebhookData struct {
	Webhook *models.Webhook `json:"webhook"`
} //	@name	GetUserWebhookData

type GetUserWebhookResponse struct {
	Status string             `json:"status"`
	Data   GetUserWebhookData `json:"data"`
} //	@name	GetUserWebhookResponse

// GetUserWebhook godoc
//
//	@Summary			Get user's webhook by id
//	@Description		Retrieve webhook details by id.
//	@Id					GET-Webhook
//	@Tags				webhook
//	@Accept				json
//	@Accept				x-www-form-urlencoded
//	@Produce			json
//	@Security			BasicAuth
//	@Security			Bearer
//	@Param				id	path		string	true	"webhook's id"
//	@Success			200	{object}	GetUserWebhookResponse
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
//	@Router				/webhooks/{id} [get]
//	@x-client-action	"get"
func (c *WebhookController) GetUserWebhook(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetWebhookDetail").
			Observe(time.Since(t).Seconds())
	}()

	webhookId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Webhook's id is invalid.")
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	result, err := c.webhookService.GetWebhookById(
		ctx.Request().Context(),
		authInfo.User.Id,
		webhookId,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}
	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		GetUserWebhookData{
			Webhook: result,
		},
	)
}

// DeleteWebhook godoc
//
//	@Summary						Delete webhook
//	@Description					This endpoint will delete the indicated webhook.
//	@Tags							webhook
//	@Id								DELETE-Webhook
//	@Produce						json
//	@Security						BasicAuth
//	@Security						Bearer
//	@Param							id	path		string	true	"Webhook ID"
//
//	@Success						200	{object}	models.ResponseSuccess
//	@Header							200	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header							200	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header							200	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure						400	{object}	models.ResponseError
//	@Header							400	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header							400	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header							400	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure						403	{object}	models.ResponseError
//	@Header							403	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header							403	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header							403	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure						404	{object}	models.ResponseError
//	@Header							404	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header							404	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header							404	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure						500	{object}	models.ResponseError
//	@Header							500	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header							500	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header							500	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"//	@Router	/webhooks/{id}	[delete]
//	@x-client-copy-from-response	true
//	@x-client-description			{"default":"This method will delete the indicated webhook."}
//	@x-client-action				"delete"
func (c *WebhookController) DeleteWebhook(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteWebhookById").
			Observe(time.Since(t).Seconds())
	}()

	webhookId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Webhook's id is invalid.")
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	if err := c.webhookService.DeleteUserWebhook(
		ctx.Request().Context(),
		authInfo.User.Id,
		webhookId,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Deleted webhook successfully.",
	)
}

type UpdateWebhookRequest struct {
	Name                  *string `body:"name"              json:"name"`
	Url                   *string `body:"url"               json:"url"`
	FileReceived          *bool   `body:"file_received"     json:"file_received"`
	EventEncodingStarted  *bool   `body:"encoding_started"  json:"encoding_started"`
	EventEncodingFinished *bool   `body:"encoding_finished" json:"encoding_finished"`
	EventEncodingFailed   *bool   `body:"encoding_failed"   json:"encoding_failed"`
	EventPartialFinished  *bool   `body:"partial_finished"  json:"partial_finished"`
} //	@name	UpdateWebhookRequest

// UpdateWebhook godoc
//
//	@Summary			Update event webhook
//	@Description		This endpoint will update the indicated webhook.
//	@Tags				webhook
//	@Id					PATCH-Webhook
//	@Accept				json
//	@Accept				x-www-form-urlencoded
//	@Produce			json
//	@Security			BasicAuth
//	@Security			Bearer
//	@Param				request	body		UpdateWebhookRequest	true	"Update Webhook input, events example: media.encoding.quality.completed"
//	@Param				id		path		string					true	"webhook's id"
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
//	@Header				500		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"//
//	@Router				/webhooks/{id} [patch]
//	@x-client-action	"update"
func (c *WebhookController) UpdateWebhook(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UpdateWebhook").
			Observe(time.Since(t).Seconds())
	}()

	var payload UpdateWebhookRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid parameters.")
	}

	webhookId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Webhook's Id is invalid.")
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	input := models.UpdateWebhookInput{
		Id:                    webhookId,
		UserId:                authInfo.User.Id,
		Url:                   payload.Url,
		EventEncodingFinished: payload.EventEncodingFinished,
		EventEncodingStarted:  payload.EventEncodingStarted,
		EventFileReceived:     payload.FileReceived,
		EventEncodingFailed:   payload.EventEncodingFailed,
		EventPartialFinished:  payload.EventPartialFinished,
	}
	if payload.Url != nil {
		if !validate.IsValidUrl(*payload.Url) {
			return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid URL.")
		}
	}

	if payload.Name != nil {
		name := strings.TrimSpace(*payload.Name)
		if len(*payload.Name) > 100 {
			return response.ResponseFailMessage(
				ctx,
				http.StatusBadRequest,
				"Name must be less than 100 characters.",
			)
		}

		input.Name = &name
	}

	if err := c.webhookService.UpdateWebhook(
		ctx.Request().Context(),
		input,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Update webhook successfully.",
	)
}

// CheckWebhookById godoc
//
//	@Summary				Check webhook by id
//	@Description			This endpoint will check the indicated webhook.
//	@Tags					webhook
//	@Id						POST-CheckWebhook
//	@Accept					json
//	@Accept					x-www-form-urlencoded
//	@Produce				json
//	@Security				Bearer
//	@Security				BasicAuth
//	@Param					id	path		string	true	"webhook's id"
//	@Success				200	{object}	models.ResponseSuccess
//	@Header					200	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					200	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					200	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure				400	{object}	models.ResponseError
//	@Header					400	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					400	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					400	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure				403	{object}	models.ResponseError
//	@Header					403	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					403	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					403	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure				404	{object}	models.ResponseError
//	@Header					404	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					404	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					404	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure				500	{object}	models.ResponseError
//	@Header					500	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header					500	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header					500	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router					/webhooks/check/{id} [post]
//	@x-client-action		"check"
//	@x-client-description	{"default":"This method will check the indicated webhook."}
func (c *WebhookController) CheckWebhookById(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CheckWebhookById").
			Observe(time.Since(t).Seconds())
	}()

	webhookId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Webhook's Id is invalid.")
	}
	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	if err := c.webhookService.CheckWebhookById(
		ctx.Request().Context(),
		authInfo.User.Id,
		webhookId,
	); err != nil {
		return response.ResponseError(ctx, err)
	}
	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Check webhook successfully.",
	)
}
