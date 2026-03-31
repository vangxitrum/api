package routes

import (
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/controllers"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/middlewares"
)

type PlaylistRoute struct {
	playlistController *controllers.PlaylistController
	apiKeyAuth         echo.MiddlewareFunc
}

func NewPlaylistRoute(
	playlistController *controllers.PlaylistController,
	apiKeyAuth echo.MiddlewareFunc,
) *PlaylistRoute {
	return &PlaylistRoute{
		playlistController: playlistController,
		apiKeyAuth:         apiKeyAuth,
	}
}

func (r *PlaylistRoute) Register(rg *echo.Group) {
	playlistRoute := rg.Group("/playlists")

	authRg := playlistRoute.Group("")
	authRg.Use(
		middlewares.NewRateLimiter(
			"/playlists"),
	)
	authRg.Use(r.apiKeyAuth)

	authRg.GET(
		"/:id",
		r.playlistController.GetPlaylistById,
	)

	authRg.POST(
		"/create",
		r.playlistController.CreatePlaylist,
	)

	authRg.PATCH(
		"/:id",
		r.playlistController.UpdatePlaylist,
	)

	authRg.POST(
		"",
		r.playlistController.GetUserPlaylists,
	)

	authRg.POST(
		"/:id/items",
		r.playlistController.AddMediaToPlaylist,
	)

	authRg.PUT(
		"/:id/items",
		r.playlistController.MoveMediaInPlaylist,
	)

	authRg.DELETE(
		"/:id",
		r.playlistController.DeletePlaylistById,
	)

	authRg.DELETE(
		"/:id/items/:item_id",
		r.playlistController.RemoveMediaFromPlaylist,
	)

	authRg.DELETE(
		"/:id/thumbnail",
		r.playlistController.DeletePlaylistThumbnail,
	)

	noAuthRg := playlistRoute.Group("")
	noAuthRg.GET(
		"/:id/thumbnail",
		r.playlistController.GetPlaylistThumbnail,
	)

	noAuthRg.GET(
		"/:id/player.json",
		r.playlistController.GetPlaylistPublic,
	)
}
