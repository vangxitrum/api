package controllers

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/slice"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/validate"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

type PlaylistController struct {
	playlistService *services.PlaylistService
	mediaService    *services.MediaService
	usageService    *services.UsageService
}

func NewPlaylistController(
	playlistService *services.PlaylistService,
	mediaService *services.MediaService,
	usageService *services.UsageService,
) *PlaylistController {
	return &PlaylistController{
		playlistService: playlistService,
		mediaService:    mediaService,
		usageService:    usageService,
	}
}

type CreatePlaylistRequest struct {
	Name         string            `json:"name"          form:"name"`
	PlaylistType string            `json:"playlist_type" form:"playlist_type"`
	Metadata     []models.Metadata `json:"metadata"      form:"metadata"`
	Tags         []string          `json:"tags"          form:"tags"`
} //	@name	CreatePlaylistRequest

type CreatePlaylistData struct {
	Playlist *models.Playlist `json:"playlist"`
} //	@name	CreatePlaylistData

type CreatePlaylistResponse struct {
	Status string             `json:"status"`
	Data   CreatePlaylistData `json:"data"`
} //	@name	CreatePlaylistResponse

// CreatePlaylist godoc
//
//	@Summary		Create a new playlist
//	@Description	Create a new playlist for the authenticated user
//	@Tags			playlist
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BasicAuth
//	@Security		Bearer
//	@Param			body		formData	CreatePlaylistRequest	true	"Playlist information"
//	@Param			metadata	formData	[]string				false	"Playlist metadata"
//	@Param			file		formData	file					false	"Thumbnail file for the playlist"
//	@Success		201			{object}	CreatePlaylistResponse
//	@Failure		400			{object}	models.ResponseError
//	@Failure		401			{object}	models.ResponseError
//	@Failure		500			{object}	models.ResponseError
//	@Router			/playlists/create [post]
func (c *PlaylistController) CreatePlaylist(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CreatePlaylist").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	var payload CreatePlaylistRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	words := strings.Fields(payload.Name)
	payload.Name = strings.Join(words, " ")

	if payload.Name == "" {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Name is required.")
	}

	if len(payload.Name) > models.TitleMaxLen {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			fmt.Sprintf("Name must be less than %d characters.", models.TitleMaxLen),
		)
	}

	if len(payload.Tags) > 0 {
		if err := validate.IsValidateTags(&payload.Tags); err != nil {
			return response.ResponseError(ctx, err)
		}
	}

	if len(payload.Metadata) > 0 {
		if err := validate.IsValidateMetadata(&payload.Metadata); err != nil {
			return response.ResponseError(ctx, err)
		}
	}

	if payload.PlaylistType == "" {
		payload.PlaylistType = models.VideoType
	} else if payload.PlaylistType != models.VideoType && payload.PlaylistType != models.AudioType {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid media type.")
	}

	playlist, err := c.playlistService.CreatePlaylist(
		ctx.Request().Context(),
		authInfo.User.Id,
		payload.Name,
		payload.Metadata,
		payload.Tags,
		payload.PlaylistType,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusCreated,
		CreatePlaylistData{
			Playlist: playlist,
		},
	)
}

type GetPlaylistByIdRequest struct {
	SortBy  string `json:"sort_by"  form:"sort_by"  query:"sort_by"`
	OrderBy string `json:"order_by" form:"order_by" query:"order_by"`
	Search  string `json:"search"   form:"search"   query:"search"`
} //	@name	GetPlaylistByIdRequest

type GetPlaylistByIdData struct {
	Playlist *models.Playlist `json:"playlist"`
} //	@name	GetPlaylistByIdData

type GetPlaylistByIdResponse struct {
	Status string              `json:"status"`
	Data   GetPlaylistByIdData `json:"data"`
} //	@name	GetPlaylistByIdResponse

// GetPlaylistById godoc
//
//	@Summary		Get playlist by ID
//	@Description	Retrieve a specific playlist by its ID for the current user.
//	@Tags			playlist
//	@Accept			json
//	@Produce		json
//	@Security		BasicAuth
//	@Security		Bearer
//	@Param			id			path		string	true	"Playlist ID"
//	@Param			sort_by		query		string	false	"Sort by field (created_at, title, duration)"
//	@Param			order_by	query		string	false	"Order by (asc, desc)"
//	@Param			search		query		string	false	"Search term"
//	@Success		200			{object}	GetPlaylistByIdResponse
//	@Failure		400			{object}	models.ResponseError
//	@Failure		403			{object}	models.ResponseError
//	@Failure		404			{object}	models.ResponseError
//	@Failure		500			{object}	models.ResponseError
//	@Router			/playlists/{id} [get]
func (c *PlaylistController) GetPlaylistById(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetPlaylistById").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	var payload GetPlaylistByIdRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	if payload.SortBy != "" && !models.PlaylistSortByMap[payload.SortBy] {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Allowed values: created_at, name.",
		)
	}

	if payload.OrderBy != "" && !models.OrderMap[payload.OrderBy] {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Unknown sorting order. Please use \"asc\" or \"desc\".",
		)
	}

	if payload.Search != "" {
		payload.Search = strings.TrimSpace(payload.Search)
	}

	playlistId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Playlist's Id is invalid.")
	}

	filter := &models.PlaylistItemFilter{
		SortBy:  payload.SortBy,
		OrderBy: payload.OrderBy,
		Search:  payload.Search,
	}

	playlist, err := c.playlistService.GetPlaylistById(
		ctx.Request().Context(),
		authInfo.User.Id,
		playlistId,
		filter,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		GetPlaylistByIdData{
			Playlist: playlist,
		},
	)
}

type GetPlaylistListRequest struct {
	Offset       int               `json:"offset"        form:"offset"`
	Limit        int               `json:"limit"         form:"limit"`
	SortBy       string            `json:"sort_by"       form:"sort_by"`
	Search       string            `json:"search"        form:"search"`
	OrderBy      string            `json:"order_by"      form:"order_by"`
	Metadata     []models.Metadata `json:"metadata"      form:"metadata"`
	PlaylistType string            `json:"playlist_type" form:"playlist_type"`
	Tags         []string          `json:"tags"          form:"tags"`
} //	@name	GetPlaylistListRequest

type GetPlaylistListResponse struct {
	Status string              `json:"status"`
	Data   GetPlaylistListData `json:"data"`
} //	@name	GetPlaylistListResponse

type GetPlaylistListData struct {
	Playlists []*models.Playlist `json:"playlists"`
	Total     int64              `json:"total"`
} //	@name	GetPlaylistListData

// GetUserPlaylists godoc
//
//	@Summary		Get user's playlists
//	@Description	Retrieve a list of playlists for the authenticated user
//	@Tags			playlist
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BasicAuth
//	@Security		Bearer
//	@Param			search			formData	string		false	"Search term"
//	@Param			sort_by			formData	string		false	"Sort by field (created_at, name, status)"
//	@Param			order_by		formData	string		false	"Order by (asc, desc)"
//	@Param			offset			formData	int			false	"Offset for pagination"
//	@Param			limit			formData	int			false	"Limit for pagination (max 25)"
//	@Param			playlist_type	formData	string		false	"type of playlist (audio, video)"
//	@Param			tags			formData	[]string	false	"Filter by tags"
//	@Param			metadata		formData	[]string	false	"Filter by metadata (key:value format)"
//	@Success		200				{object}	GetPlaylistListData
//	@Failure		400				{object}	models.ResponseError
//	@Failure		401				{object}	models.ResponseError
//	@Failure		403				{object}	models.ResponseError
//	@Failure		500				{object}	models.ResponseError
//	@Router			/playlists [post]
func (c *PlaylistController) GetUserPlaylists(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetUserPlaylists").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	var payload GetPlaylistListRequest

	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	if payload.Limit > int(models.MaxPageLimit) {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Limit only allowed values 1-100.",
		)
	}

	if payload.Limit > models.PageSizeLimit {
		payload.Limit = models.PageSizeLimit
	}

	if payload.SortBy != "" && !models.PlaylistSortByMap[payload.SortBy] {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Allowed values: created_at, name.",
		)
	}

	if payload.OrderBy != "" && !models.OrderMap[payload.OrderBy] {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Unknown sorting order. Please use \"asc\" or \"desc\".",
		)
	}

	if len(payload.Tags) > 0 {
		if err := validate.IsValidateTags(&payload.Tags); err != nil {
			return response.ResponseError(ctx, err)
		}
	}

	if len(payload.Metadata) > 0 {
		if err := validate.IsValidateMetadata(&payload.Metadata); err != nil {
			return response.ResponseError(ctx, err)
		}
	}

	if payload.Search != "" {
		payload.Search = strings.TrimSpace(payload.Search)
	}
	if payload.SortBy == "" {
		payload.SortBy = "created_at"
	}
	if payload.OrderBy == "" {
		payload.OrderBy = "desc"
	}

	if payload.Limit == 0 {
		payload.Limit = models.PageSizeLimit
	}

	if payload.PlaylistType != "" &&
		payload.PlaylistType != models.VideoType &&
		payload.PlaylistType != models.AudioType {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest,
			"Invalid playlist type. Must be 'video' or 'audio'.")
	}

	playlists, total, err := c.playlistService.GetUserPlaylists(
		ctx.Request().Context(),
		authInfo.User.Id,
		models.PlaylistFilter{
			Search:       payload.Search,
			SortBy:       payload.SortBy,
			OrderBy:      payload.OrderBy,
			Offset:       payload.Offset,
			Limit:        payload.Limit,
			Metadata:     payload.Metadata,
			Tags:         payload.Tags,
			PlaylistType: payload.PlaylistType,
		},
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		GetPlaylistListData{
			Playlists: playlists,
			Total:     total,
		},
	)
}

// DeletePlaylistById godoc
//
//	@Summary		Delete a playlist by ID
//	@Description	Delete a specific playlist by its ID for the authenticated user
//	@Tags			playlist
//	@Accept			json
//	@Produce		json
//	@Security		BasicAuth
//	@Security		Bearer
//	@Param			id	path		string	true	"Playlist ID"	format(uuid)
//	@Success		200	{object}	models.ResponseSuccess
//	@Failure		400	{object}	models.ResponseError
//	@Failure		401	{object}	models.ResponseError
//	@Failure		403	{object}	models.ResponseError
//	@Failure		404	{object}	models.ResponseError
//	@Failure		500	{object}	models.ResponseError
//	@Router			/playlists/{id} [delete]
func (c *PlaylistController) DeletePlaylistById(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeletePlaylistById").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	playlistId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Playlist's Id is invalid.")
	}

	if err := c.playlistService.DeletePlaylistById(
		ctx.Request().Context(),
		authInfo.User.Id,
		playlistId,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Delete playlist successfully.",
	)
}

type AddMediaToPlaylistRequest struct {
	MediaId         uuid.UUID   `json:"media_id"`
	OptionPlaylists []uuid.UUID `json:"option_playlists"`
	OptionMediaIds  []uuid.UUID `json:"option_media_ids"`
} //	@name	AddMediaToPlaylistRequest

// AddMediaToPlaylist godoc
//
//	@Summary		Add a media to a playlist
//	@Description	Add a specific media to a playlist for the authenticated user
//	@Tags			playlist
//	@Accept			json
//	@Produce		json
//	@Security		BasicAuth
//	@Security		Bearer
//	@Param			id		path		string						true	"Playlist ID"	format(uuid)
//	@Param			payload	body		AddMediaToPlaylistRequest	true	"Media details"
//	@Success		200		{object}	models.ResponseSuccess
//	@Failure		400		{object}	models.ResponseError
//	@Failure		401		{object}	models.ResponseError
//	@Failure		403		{object}	models.ResponseError
//	@Failure		404		{object}	models.ResponseError
//	@Failure		500		{object}	models.ResponseError
//	@Router			/playlists/{id}/items [post]
func (c *PlaylistController) AddMediaToPlaylist(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("AddMediaToPlaylist").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	playlistId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Playlist's Id is invalid.")
	}

	var payload AddMediaToPlaylistRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	playlistIds := []uuid.UUID{playlistId}
	if len(payload.OptionPlaylists) > 0 {
		playlistIds = append(playlistIds, payload.OptionPlaylists...)
	}

	mediaIds := []uuid.UUID{payload.MediaId}
	if len(payload.OptionMediaIds) > 0 {
		mediaIds = append(mediaIds, payload.OptionMediaIds...)
	}

	uniquePlaylistIds := slice.DeDupUUIDs(playlistIds)
	uniqueMediaIds := slice.DeDupUUIDs(mediaIds)

	if len(uniquePlaylistIds) == 0 || len(uniqueMediaIds) == 0 {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	if err := c.playlistService.AddMediaToPlaylist(
		ctx.Request().Context(),
		authInfo.User.Id,
		uniquePlaylistIds,
		uniqueMediaIds,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Add media to playlist successfully.",
	)
}

type RemoveMediasFromPlaylistRequest struct {
	OptionPlaylists []uuid.UUID `json:"option_playlists"`
} //	@name	RemoveMediasFromPlaylistRequest

// RemoveMediaFromPlaylist godoc
//
//	@Summary		Remove a media from a playlist
//	@Description	Remove a specific media from a playlist for the authenticated user
//	@Tags			playlist
//	@Accept			json
//	@Produce		json
//	@Security		BasicAuth
//	@Security		Bearer
//	@Param			id		path		string							true	"Playlist ID"		format(uuid)
//	@Param			item_id	path		string							true	"Playlist Item ID"	format(uuid)
//	@Param			payload	body		RemoveMediasFromPlaylistRequest	false	"Optional payload"
//	@Success		200		{object}	models.ResponseSuccess
//	@Failure		400		{object}	models.ResponseError
//	@Failure		401		{object}	models.ResponseError
//	@Failure		403		{object}	models.ResponseError
//	@Failure		404		{object}	models.ResponseError
//	@Failure		500		{object}	models.ResponseError
//	@Router			/playlists/{id}/items/{item_id} [delete]
func (c *PlaylistController) RemoveMediaFromPlaylist(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("MoveMediaInPlaylist").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	playlistId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Playlist's Id is invalid.")
	}

	itemId, err := uuid.Parse(ctx.Param("item_id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Media's Id is invalid.")
	}

	var payload RemoveMediasFromPlaylistRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	optionPlaylistIds := []uuid.UUID{playlistId}
	if len(payload.OptionPlaylists) > 0 {
		optionPlaylistIds = append(optionPlaylistIds, payload.OptionPlaylists...)
	}

	if err := c.playlistService.RemoveMediaFromPlaylist(
		ctx.Request().Context(),
		authInfo.User.Id,
		playlistId,
		itemId,
		optionPlaylistIds,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Media removed from playlist successfully.",
	)
}

type MoveMediaInPlaylistRequest struct {
	// CurrentId is the UUID of the playlist item (media) to be moved
	CurrentId uuid.UUID `json:"current_id" example:"123e4567-e89b-12d3-a456-426614174000"`

	// NextId is the UUID of the playlist item that should come after the moved item
	NextId *uuid.UUID `json:"next_id" example:"123e4567-e89b-12d3-a456-426614174001"`

	// PreviousId is the UUID of the playlist item that should come before the moved item
	PreviousId *uuid.UUID `json:"previous_id" example:"123e4567-e89b-12d3-a456-426614174002"`
} //	@name	MoveMediaInPlaylistRequest

// MoveMediaInPlaylist godoc
//
//	@Summary		Move a media within a playlist
//	@Description	Change the position of a media in a playlist for the authenticated user.
//	@Description
//	@Description	**Examples:**
//	@Description	1. **Move to top:**
//	@Description	```json
//	@Description	{
//	@Description	"current_id": "123e4567-e89b-12d3-a456-426614174000",
//	@Description	"next_id": "first-item-id",
//	@Description	}
//	@Description	```
//	@Description
//	@Description	2. **Move to bottom:**
//	@Description	```json
//	@Description	{
//	@Description	"current_id": "123e4567-e89b-12d3-a456-426614174000",
//	@Description	"previous_id": "last-item-id"
//	@Description	}
//	@Description	```
//	@Description	3. **Move between two items:**
//	@Description	```json
//	@Description	{
//	@Description	"current_id": "123e4567-e89b-12d3-a456-426614174000",
//	@Description	"next_id": "item-after-id",
//	@Description	"previous_id": "item-before-id"
//	@Description	}
//	@Description	```
//	@Description	> **Note:** The `current_id` is always required. Use `next_id` and `previous_id` to specify the new position.
//	@Description
//	@Description	> **Important:** If the specified position is invalid (e.g., wrong position of `next_id` or `previous_id`),
//	@Description	> the operation will fail and return an error. The playlist will remain unchanged in such cases.
//	@Tags			playlist
//	@Accept			json
//	@Produce		json
//	@Security		BasicAuth
//	@Security		Bearer
//	@Param			id		path		string						true	"Playlist ID"	format(uuid)
//	@Param			payload	body		MoveMediaInPlaylistRequest	true	"Move media details"
//	@Success		200		{object}	models.ResponseSuccess
//	@Failure		400		{object}	models.ResponseError
//	@Failure		401		{object}	models.ResponseError
//	@Failure		403		{object}	models.ResponseError
//	@Failure		404		{object}	models.ResponseError
//	@Failure		500		{object}	models.ResponseError
//	@Router			/playlists/{id}/items/ [put]
func (c *PlaylistController) MoveMediaInPlaylist(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("MoveMediaInPlaylist").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	playlistId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Playlist's Id is invalid.")
	}

	var payload MoveMediaInPlaylistRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	if payload.CurrentId == uuid.Nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Current Id is invalid.")
	}
	var nextId, previousId *uuid.UUID
	if payload.NextId != nil && *payload.NextId != uuid.Nil {
		nextId = payload.NextId
	}
	if payload.PreviousId != nil && *payload.PreviousId != uuid.Nil {
		previousId = payload.PreviousId
	}

	err = c.playlistService.MoveItemInPlaylist(
		ctx.Request().Context(),
		authInfo.User.Id,
		playlistId,
		payload.CurrentId,
		nextId,
		previousId,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Media moved successfully in playlist.",
	)
}

type UpdatePlaylistRequest struct {
	Name     *string           `json:"name"     form:"name"`
	Tags     []string          `json:"tags"     form:"tags"`
	Metadata []models.Metadata `json:"metadata" form:"metadata"`
} //	@name	UpdatePlaylistRequest

// UpdatePlaylist godoc
//
//	@Summary		Update a playlist
//	@Description	Update details of a specific playlist for the authenticated user
//	@Tags			playlist
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BasicAuth
//	@Security		Bearer
//	@Param			id			path		string		true	"Playlist ID"	format(uuid)
//	@Param			name		formData	string		false	"Playlist name"
//	@Param			tags		formData	[]string	false	"Playlist tags"
//	@Param			metadata	formData	[]string	false	"Playlist metadata"
//	@Param			file		formData	file		false	"New thumbnail file for the playlist"
//	@Success		200			{object}	models.ResponseSuccess
//	@Failure		400			{object}	models.ResponseError
//	@Failure		401			{object}	models.ResponseError
//	@Failure		403			{object}	models.ResponseError
//	@Failure		404			{object}	models.ResponseError
//	@Failure		500			{object}	models.ResponseError
//	@Router			/playlists/{id} [patch]
func (c *PlaylistController) UpdatePlaylist(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UpdatePlaylist").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)

	playlistId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Playlist's Id is invalid.")
	}

	var payload UpdatePlaylistRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	if payload.Name != nil {
		words := strings.Fields(*payload.Name)
		*payload.Name = strings.Join(words, " ")
		if len(*payload.Name) > models.TitleMaxLen {
			return response.ResponseFailMessage(
				ctx,
				http.StatusBadRequest,
				fmt.Sprintf("Name must be less than %d characters.", models.TitleMaxLen),
			)
		}
	}

	if len(payload.Tags) > 0 {
		if err := validate.IsValidateTags(&payload.Tags); err != nil {
			return response.ResponseError(ctx, err)
		}
	}

	if len(payload.Metadata) > 0 {
		if err := validate.IsValidateMetadata(&payload.Metadata); err != nil {
			return response.ResponseError(ctx, err)
		}
	}

	var thumbnailFile *multipart.FileHeader
	file, err := ctx.FormFile("file")
	if err == nil {
		if err := validate.IsValidateThumbnail(file); err != nil {
			return err
		}
		thumbnailFile = file
	} else if !errors.Is(err, http.ErrMissingFile) {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid thumbnail.")
	}

	if err := c.playlistService.UpdatePlaylistInfo(
		ctx.Request().Context(),
		authInfo.User.Id,
		models.UpdatePlaylistInput{
			PlaylistId: playlistId,
			Name:       payload.Name,
			Metadata:   payload.Metadata,
			Tags:       payload.Tags,
			Thumbnail:  thumbnailFile,
		},
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Update playlist successfully.",
	)
}

func (c *PlaylistController) GetPlaylistThumbnail(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetPlaylistThumbnail").
			Observe(time.Since(t).Seconds())
	}()
	playlistId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Playlist's id is invalid.")
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

	fileInfo, err := c.playlistService.GetPlaylistThumbnail(
		ctx.Request().Context(),
		playlistId,
		resolution,
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

// GetPlaylistPublic godoc
//
//	@Summary		Get a playlist public
//	@Description	Get a specific playlist public by its ID
//	@Tags			playlist
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Playlist ID"	format(uuid)
//	@Success		200	{object}	models.PublicPlaylistObject
//	@Failure		400	{object}	models.ResponseError
//	@Failure		500	{object}	models.ResponseError
//	@Router			/playlists/{id}/public [get]
func (c *PlaylistController) GetPlaylistPublic(ctx echo.Context) error {
	playlistId, err := uuid.Parse(ctx.Param("id"))
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

	playlist, playerTheme, err := c.playlistService.GetPlaylistPublicById(
		ctx.Request().Context(),
		playlistId,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return ctx.JSON(http.StatusOK, models.NewPlaylistPublicObject(playlist, playerTheme))
}

// DeletePlaylistThumbnail godoc
//
//	@Summary		Delete a playlist thumbnail
//	@Description	Delete the thumbnail of a specific playlist for the authenticated user
//	@Tags			playlist
//	@Accept			json
//	@Produce		json
//	@Security		BasicAuth
//	@Security		Bearer
//	@Param			id	path		string	true	"Playlist ID"	format(uuid)
//	@Success		200	{object}	models.ResponseSuccess
//	@Failure		400	{object}	models.ResponseError
//	@Failure		401	{object}	models.ResponseError
//	@Failure		403	{object}	models.ResponseError
//	@Failure		404	{object}	models.ResponseError
//	@Failure		500	{object}	models.ResponseError
//	@Router			/playlists/{id}/thumbnail [delete]
func (c *PlaylistController) DeletePlaylistThumbnail(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeletePlaylistThumbnail").
			Observe(time.Since(t).Seconds())
	}()
	userId := ctx.Get("authInfo").(models.AuthenticationInfo).User.Id
	playlistId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Playlist's id is invalid.")
	}
	if err := c.playlistService.DeletePlaylistThumbnail(
		ctx.Request().Context(),
		userId,
		playlistId,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Deleted playlist thumbnail successfully.",
	)
}
