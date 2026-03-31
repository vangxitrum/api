package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	utils "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/payload"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

type ApiKeyController struct {
	apiKeyService *services.ApiKeyService
}

func NewApiKeyController(apiKeyService *services.ApiKeyService) *ApiKeyController {
	return &ApiKeyController{
		apiKeyService: apiKeyService,
	}
}

type CreateApiKeyRequest struct {
	Name string `json:"api_key_name"`
	Type string `json:"type"`
	TTL  string `json:"ttl"`
} //	@name	CreateApiKeyRequest

type CreateApiKeyData struct {
	APIKey *models.ApiKey `json:"api_key"`
} //	@name	CreateApiKeyData

type CreateApiKeyResponse struct {
	Status string           `json:"status"`
	Data   CreateApiKeyData `json:"data"`
} //	@name	CreateApiKeyResponse

// CreateApiKey godoc
//
//	@Summary			Create API key
//	@Description		This endpoint enables you to create a new API key for a specific project.
//	@Id					CREATE-api-key
//	@Tags				apiKey
//	@Accept				json
//	@Accept				x-www-form-urlencoded
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				request	body		CreateApiKeyRequest	true	"api key's data"
//	@Success			201		{object}	CreateApiKeyResponse
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
//	@Router				/api_keys [post]
//	@x-client-action	"create"
func (c *ApiKeyController) CreateApiKey(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CreateApiKey").Observe(time.Since(t).Seconds())
	}()

	var payload CreateApiKeyRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid request payload.")
	}

	if strings.TrimSpace(payload.Name) == "" {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid API key name.")
	}

	if len(payload.Name) > models.NameLength {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"API key name is too long.",
		)
	}

	numTtl, err := strconv.ParseInt(
		payload.TTL,
		10,
		64,
	)
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid ttl value.")
	}
	if numTtl > models.MaxTtl {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Maximum ttl value is 2147483647.",
		)
	}
	if numTtl == models.MinTtl {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "TTL must be specified.")
	}
	if numTtl < models.MinTtl {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid ttl value.")
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	if (payload.Type != string(models.FullAccess) && payload.Type != string(models.OnlyUpload)) ||
		payload.Type == "" {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid api key type.")
	}

	result, err := c.apiKeyService.CreateApiKey(
		ctx.Request().Context(),
		authInfo.User.Id,
		strings.TrimSpace(payload.Name),
		payload.TTL,
		payload.Type,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusCreated,
		CreateApiKeyData{
			APIKey: result,
		},
	)
}

type GetApiKeysRequest struct {
	Search  string `json:"search"   form:"search"   query:"search"`
	SortBy  string `json:"sort_by"  form:"sort_by"  query:"sort_by"`
	OrderBy string `json:"order_by" form:"order_by" query:"order_by"`
	Offset  uint64 `json:"offset"   form:"offset"   query:"offset"`
	Limit   uint64 `json:"limit"    form:"limit"    query:"limit"`
	Type    string `json:"type"     form:"type"     query:"type"`
}

type GetApiKeysData struct {
	APIKeys []*models.ApiKey `json:"api_keys"`
	Total   int64            `json:"total"`
} //	@name	GetApiKeysData

type GetApiKeysResponse struct {
	Status string         `json:"status"`
	Data   GetApiKeysData `json:"data"`
} //	@name	GetApiKeysResponse

// GetApiKeyList godoc
//
//	@Summary			Get list API keys
//	@Description		Retrieve a list of all API keys for the current workspace.
//	@Tags				apiKey
//	@Id					GET-api-keys
//	@Accept				json
//	@Accept				x-www-form-urlencoded
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				search		query		string	false	"only support search by name"
//	@Param				sort_by		query		string	false	"sort by"														Enums(created_at, name)	default(created_at)
//	@Param				order_by	query		string	false	"allowed: asc, desc. Default: asc"								Enums(asc,desc)			default(asc)
//	@Param				offset		query		integer	false	"offset, allowed values greater than or equal to 0. Default(0)"	minimum(0)				default(0)
//	@Param				limit		query		integer	false	"results per page. Allowed values 1-100, default is 25"			minimum(1)				maximum(100)	default(25)
//	@Success			200			{object}	GetApiKeysResponse
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
//	@Router				/api_keys [get]
//	@x-group-parameters	true
//	@x-client-paginated	true
//	@x-optional-object	true
//	@x-client-action	"list"
func (c *ApiKeyController) GetApiKeyList(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetApiKeyList").
			Observe(time.Since(t).Seconds())
	}()
	var payload GetApiKeysRequest
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

	if payload.Type != "" {
		if payload.Type != string(models.FullAccess) && payload.Type != string(models.OnlyUpload) {
			return response.ResponseFailMessage(
				ctx,
				http.StatusBadRequest,
				"Unknown access type. Please use \"full_access\" or \"only_upload\".",
			)
		}
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
	filter := models.GetApiKeyListInput{
		UserId: authInfo.User.Id,
		Search: payload.Search,
		SortBy: filterPayload.SortBy,
		Order:  filterPayload.Order,
		Offset: payload.Offset,
		Limit:  filterPayload.Limit,
		Type:   payload.Type,
	}
	result, total, err := c.apiKeyService.GetApiKeyList(
		ctx.Request().Context(),
		filter,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx, http.StatusOK, GetApiKeysData{
			APIKeys: result,
			Total:   total,
		},
	)
}

// DeleteApiKey godoc
//
//	@Summary			Delete API key
//	@Description		This endpoint enables you to delete an API key from a specific project.
//	@Tags				apiKey
//	@Id					DELETE-api-key
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id	path		string	true	"API key's ID"
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
//	@Router				/api_keys/{id} [delete]
//	@x-client-action	"delete"
func (c *ApiKeyController) DeleteApiKey(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteApiKey").Observe(time.Since(t).Seconds())
	}()

	apiKeyId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "API key's Id is invalid.")
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	err = c.apiKeyService.DeleteUserApiKey(
		ctx.Request().Context(),
		apiKeyId,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Delete api key successfully.",
	)
}

type RenameAPIKeyRequest struct {
	Name string `json:"api_key_name"`
} //	@name	RenameAPIKeyRequest

// UpdateApiKey godoc
//
//	@Summary			Rename api key
//	@Description		This endpoint enables you to rename an API key from a specific project.
//	@Tags				apiKey
//	@Id					PATCH-api-key
//	@Accept				json
//	@Accept				x-www-form-urlencoded
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id		path		string				true	"api key id"
//	@Param				request	body		RenameAPIKeyRequest	true	"new api key name"
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
//	@Router				/api_keys/{id} [patch]
//	@x-client-action	"update"
func (c *ApiKeyController) UpdateApiKey(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UpdateApiKey").Observe(time.Since(t).Seconds())
	}()
	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	var payload RenameAPIKeyRequest

	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid parameters.")
	}

	if len(payload.Name) > models.NameLength {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"API key name is too long.",
		)
	}

	apiKeyId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "API key's Id is invalid.")
	}

	if payload.Name == "" {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid API key name.")
	}

	if err := c.apiKeyService.ChangeApiKey(
		ctx.Request().Context(),
		apiKeyId,
		authInfo.User.Id,
		strings.TrimSpace(payload.Name),
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Rename api keys successfully.",
	)
}
