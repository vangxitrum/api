package controllers

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	imageHelper "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/image"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	utils "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/payload"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/validate"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

type PlayerThemesController struct {
	playerThemeService *services.PlayerThemeService
	usageService       *services.UsageService
}

func NewPlayerThemesController(
	playerThemeService *services.PlayerThemeService,
	usageService *services.UsageService,
) *PlayerThemesController {
	return &PlayerThemesController{
		playerThemeService: playerThemeService,
		usageService:       usageService,
	}
}

type CreatePlayerThemeRequest struct {
	Name      string           `json:"name"`
	Theme     models.Theme     `json:"theme,omitempty"`
	Controls  *models.Controls `json:"controls,omitempty"`
	IsDefault *bool            `json:"is_default"`
} //	@name	CreatePlayerThemeRequest

type CreatePlayerThemesData struct {
	PlayerTheme *models.PlayerTheme `json:"player_theme"`
} //	@name	CreatePlayerThemesData

type CreatePlayerThemesResponse struct {
	Status string                 `json:"status"`
	Data   CreatePlayerThemesData `json:"data"`
} //	@name	CreatePlayerThemesResponse

// CreatePlayerTheme godoc
//
//	@Summary			Create a player theme
//	@Description		Create a player for your media, and customize it.
//	@Tags				players
//	@Id					POST_players
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				request	body		CreatePlayerThemeRequest	true	"Player theme input"
//	@Success			201		{object}	CreatePlayerThemesResponse
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
//	@Router				/players [post]
//	@x-client-action	"create"
func (c *PlayerThemesController) CreatePlayerTheme(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CreatePlayerTheme").
			Observe(time.Since(t).Seconds())
	}()
	var payload models.PlayerThemeInput
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid request payload.")
	}

	if !validate.AreThemeColorsValid(payload.Theme) {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid theme colors.")
	}

	if !validate.AreThemePixelsValid(payload.Theme) {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid theme pixels.")
	}

	if len(payload.Name) > models.NameLength {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Name is too long.")
	}

	trimmedName := strings.TrimSpace(payload.Name)
	if trimmedName == "" {
		trimmedName = "default-player-theme"
	}
	payload.Name = trimmedName

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	result, err := c.playerThemeService.CreatePlayerTheme(
		ctx.Request().Context(),
		authInfo.User.Id,
		payload,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusCreated,
		CreatePlayerThemesData{
			result,
		},
	)
}

type GetPlayerThemeRequest struct {
	Search  string `json:"search"   form:"search"   query:"search"`
	SortBy  string `json:"sort_by"  form:"sort_by"  query:"sort_by"`
	OrderBy string `json:"order_by" form:"order_by" query:"order_by"`
	Offset  uint64 `json:"offset"   form:"offset"   query:"offset"`
	Limit   uint64 `json:"limit"    form:"limit"    query:"limit"`
}

type GetPlayerThemeData struct {
	PlayerThemes []*models.PlayerTheme `json:"player_themes"`
	Total        int64                 `json:"total"`
} //	@name	GetPlayerThemeData

type GetPlayerThemeResponse struct {
	Status string             `json:"status"`
	Data   GetPlayerThemeData `json:"data"`
} //	@name	GetPlayerThemeResponse

// ListAllPlayersThemes godoc
//
//	@Summary			List all player themes
//	@Description		Retrieve a list of all the player themes you created, as well as details about each one.
//	@Tags				players
//	@Id					GET_players
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				search		query		string	false	"only support search by name"
//	@Param				sort_by		query		string	false	"sort by"														Enums(created_at, name)	default(created_at)
//	@Param				order_by	query		string	false	"allowed: asc, desc. Default: asc"								Enums(asc,desc)			default(asc)
//	@Param				offset		query		integer	false	"offset, allowed values greater than or equal to 0. Default(0)"	minimum(0)				default(0)
//	@Param				limit		query		integer	false	"results per page. Allowed values 1-100, default is 25"			minimum(1)				maximum(100)	default(25)
//	@Success			200			{object}	GetPlayerThemeResponse
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
//	@Router				/players [get]
//	@x-group-parameters	true
//	@x-client-paginated	true
//	@x-optional-object	true
//	@x-client-action	"list"
func (c *PlayerThemesController) ListAllPlayersThemes(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("ListAllPlayersThemes").
			Observe(time.Since(t).Seconds())
	}()
	var payload GetPlayerThemeRequest
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
	filter := models.GetThemePlayerList{
		UserId: authInfo.User.Id,
		SortBy: filterPayload.SortBy,
		Search: payload.Search,
		Order:  filterPayload.Order,
		Offset: payload.Offset,
		Limit:  filterPayload.Limit,
	}
	result, total, err := c.playerThemeService.ListAllPlayersThemes(
		ctx.Request().Context(),
		filter,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		GetPlayerThemeData{
			PlayerThemes: result,
			Total:        total,
		},
	)
}

type GetPlayerThemeByIdData struct {
	PlayerTheme *models.PlayerTheme `json:"player_theme"`
} //	@name	GetPlayerThemeByIdData

type GetPlayerThemeByIdResponse struct {
	Status string                 `json:"status"`
	Data   GetPlayerThemeByIdData `json:"data"`
} //	@name	GetPlayerThemeByIdResponse

// RetrievePlayerThemeById godoc
//
//	@Summary			Get a player theme by ID
//	@Description		Retrieve a player theme by its ID, as well as details about it.
//	@Tags				players
//	@Id					GET_players-playerId
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id	path		string	true	"Player theme ID"
//	@Success			200	{object}	GetPlayerThemeByIdResponse
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
//	@Router				/players/{id} [get]
//	@x-client-action	"get"
func (c *PlayerThemesController) RetrievePlayerThemeById(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("RetrievePlayerThemeById").
			Observe(time.Since(t).Seconds())
	}()
	playerThemeId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Player theme's id is invalid.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	result, err := c.playerThemeService.GetPlayerThemeById(
		ctx.Request().Context(),
		authInfo.User.Id,
		playerThemeId,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}
	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		GetPlayerThemeByIdData{
			result,
		},
	)
}

// DeletePlayerThemeById godoc
//
//	@Summary			Delete a player theme by ID
//	@Description		Delete a player if you no longer need it. You can delete any player that you have the player ID for.
//	@Tags				players
//	@Id					DELETE_players-playerId
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id	path		string	true	"Player theme ID"
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
//	@Router				/players/{id} [delete]
//	@x-client-action	"delete"
func (c *PlayerThemesController) DeletePlayerThemeById(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeletePlayerThemeById").
			Observe(time.Since(t).Seconds())
	}()
	playerThemeId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Player theme's id is invalid.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	if err := c.playerThemeService.DeleteUserPlayerThemeById(
		ctx.Request().Context(),
		authInfo.User.Id,
		playerThemeId,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Deleted themes successfully.",
	)
}

type UpdatePlayerThemeRequest struct {
	Name      string           `json:"name"`
	Theme     models.Theme     `json:"theme,omitempty"`
	Controls  *models.Controls `json:"controls,omitempty"`
	IsDefault *bool            `json:"is_default"`
} //	@name	UpdatePlayerThemeRequest

type UpdatePlayerThemeResponse struct {
	Status string                   `json:"status"`
	Data   UpdatePlayerThemeRequest `json:"data"`
} //	@name	UpdatePlayerThemeResponse

// UpdatePlayerThemeById godoc
//
//	@Summary			Update a player theme by ID
//	@Description		Use a player ID to update specific details for a player.
//	@Tags				players
//	@Id					PATCH_players-playerId
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id		path		string						true	"Player theme ID"
//	@Param				input	body		UpdatePlayerThemeRequest	true	"Player theme input"
//	@Success			200		{object}	UpdatePlayerThemeResponse
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
//	@Router				/players/{id} [patch]
//	@x-client-action	"update"
func (c *PlayerThemesController) UpdatePlayerThemeById(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UpdatePlayerThemeById").
			Observe(time.Since(t).Seconds())
	}()
	var payload models.PlayerThemeInput

	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid parameters.")
	}

	if !validate.AreThemeColorsValid(payload.Theme) {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid theme colors.")
	}

	if !validate.AreThemePixelsValid(payload.Theme) {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid theme pixels.")
	}

	if len(payload.Name) > models.NameLength {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Name is too long.")
	}

	if payload.Name != "" {
		payload.Name = strings.TrimSpace(payload.Name)
	}

	playerThemeId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Player theme's id is invalid.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	result, err := c.playerThemeService.UpdatePlayerById(
		ctx.Request().Context(),
		playerThemeId,
		authInfo.User.Id,
		payload,
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

type UploadLogoByIdRequest struct {
	Link string                `json:"link" binding:"required"`                                                // The link to the logo (optional if a file is provided)
	File *multipart.FileHeader `json:"file" binding:"required" swaggertype:"primitive,string" format:"binary"` // The uploaded file (JPG or PNG)
} //	@name	UploadLogoByIdRequest

type UploadLogoByIdResponse struct {
	Status string              `json:"status"`
	Data   *models.PlayerTheme `json:"data"`
} //	@name	UploadLogoByIdResponse

// UploadLogoById godoc
//
//	@Summary			Upload a logo for a player theme by ID
//	@Description		Upload a logo for a player theme by its ID.
//	@Id					POST_players-playerId-logo
//	@Tags				players
//	@Accept				multipart/form-data
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id		path		string					true	"Player theme ID"
//	@Param				payload	body		UploadLogoByIdRequest	true	"Upload logo request"
//	@Success			200		{object}	UploadLogoByIdResponse
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
//	@Router				/players/{id}/logo [post]
//	@x-client-action	"uploadLogo"
func (c *PlayerThemesController) UploadLogoById(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UploadLogoById").
			Observe(time.Since(t).Seconds())
	}()
	userId := ctx.Get("authInfo").(models.AuthenticationInfo).User.Id
	playerThemeId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Player theme's id is invalid.",
		)
	}
	link := ctx.FormValue("link")
	if link != "" {
		if !validate.IsHttpsUrl(link) {
			return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid URL.")
		}
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "file logo is required.")
	}
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
	if file.Size > models.MaxLogoSize {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"The file size should be a maximum of 100 KiB.",
		)
	}
	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return response.NewInternalServerError(err)
	}
	img, _, err := image.Decode(src)
	if err != nil {
		return response.NewInternalServerError(err)
	}
	if (img.Bounds().Dx() > models.MaxLogoWidth) || (img.Bounds().Dy() > models.MaxLogoHeight) {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"The image size should be a maximum of 200px width x 100px height.",
		)
	}

	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	result, err := c.playerThemeService.UploadPlayerThemeLogo(
		ctx.Request().Context(),
		playerThemeId, userId, link,
		src,
	)
	if err != nil {
		return response.NewInternalServerError(err)
	}
	return response.ResponseSuccess(
		ctx,
		http.StatusCreated,
		result,
	)
}

// DeleteLogoById godoc
//
//	@Summary			Delete a logo for a player theme by ID
//	@Description		Delete the logo associated to a player.
//	@Tags				players
//	@Id					DELETE_players-playerId-logo
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				id	path		string	true	"Player theme ID"
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
//	@Router				/players/{id}/logo [delete]
//	@x-client-action	"deleteLogo"
func (c *PlayerThemesController) DeleteLogoById(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteLogoById").
			Observe(time.Since(t).Seconds())
	}()
	playerThemeId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Player theme's id is invalid.",
		)
	}
	userId := ctx.Get("authInfo").(models.AuthenticationInfo).User.Id
	if err := c.playerThemeService.DeletePlayerThemeLogo(
		ctx.Request().Context(),
		userId,
		playerThemeId,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Deleted logo successfully.",
	)
}

type AddPlayerThemesToMediaRequest struct {
	MediaId       uuid.UUID `json:"media_id"`
	PlayerThemeId uuid.UUID `json:"player_theme_id"`
} //	@name	AddPlayerThemesToMediaRequest

// AddPlayerThemesToMedia godoc
//
//	@Summary			Add a player theme to a media
//	@Description		Add a player theme to a media by Id.
//	@Tags				players
//	@Id					POST_players-add-player
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				request	body		AddPlayerThemesToMediaRequest	true	"Add player theme to media request"
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
//	@Router				/players/add-player [post]
//	@x-client-action	"addPlayer"
func (c *PlayerThemesController) AddPlayerThemesToMedia(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("AddPlayerThemesToMedia").
			Observe(time.Since(t).Seconds())
	}()
	var payload AddPlayerThemesToMediaRequest

	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid request payload.")
	}
	userId := ctx.Get("authInfo").(models.AuthenticationInfo).User.Id
	if err := c.playerThemeService.AddPlayerThemeToMediaById(
		ctx.Request().Context(),
		userId,
		payload.MediaId,
		payload.PlayerThemeId,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Add themes successful.",
	)
}

type RemovePlayerThemesFromMediaRequest struct {
	MediaId       uuid.UUID `json:"media_id"`
	PlayerThemeId uuid.UUID `json:"player_theme_id"`
} //	@name	RemovePlayerThemesFromMediaRequest

// RemovePlayerThemesFromMedia godoc
//
//	@Summary			Remove a player theme from a media
//	@Description		Remove a player theme from a media by Id.
//	@Tags				players
//	@Id					POST_players-remove-player
//	@Accept				json
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				request	body		RemovePlayerThemesFromMediaRequest	true	"Remove player theme from media request"
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
//	@Router				/players/remove-player [post]
//	@x-client-action	"removePlayer"
func (c *PlayerThemesController) RemovePlayerThemesFromMedia(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("RemovePlayerThemesFromMedia").
			Observe(time.Since(t).Seconds())
	}()
	var payload RemovePlayerThemesFromMediaRequest

	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid request payload.")
	}
	userId := ctx.Get("authInfo").(models.AuthenticationInfo).User.Id
	if err := c.playerThemeService.RemovePlayerThemeFromMediaById(
		ctx.Request().Context(),
		userId,
		payload.MediaId,
		payload.PlayerThemeId,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Remove themes successful.",
	)
}

func (c *PlayerThemesController) GetPlayerThemeLogo(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetPlayerThemeLogo").
			Observe(time.Since(t).Seconds())
	}()
	playerThemeId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Player theme's id is invalid.",
		)
	}

	fileInfo, err := c.playerThemeService.GetPlayerThemeLogo(
		ctx.Request().Context(),
		playerThemeId,
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
