package routes

import (
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/controllers"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/middlewares"
	"github.com/labstack/echo/v4"
)

type PlayerThemesRoute struct {
	playerThemesController *controllers.PlayerThemesController
	apiKeyAuth             echo.MiddlewareFunc
}

func NewPlayerThemesRoute(
	playerThemesController *controllers.PlayerThemesController,
	playerAuth echo.MiddlewareFunc,
) *PlayerThemesRoute {
	return &PlayerThemesRoute{
		playerThemesController: playerThemesController,
		apiKeyAuth:             playerAuth,
	}
}

func (r *PlayerThemesRoute) Register(rg *echo.Group) {
	playerRoute := rg.Group("/players")

	noAuthRg := playerRoute.Group("")
	noAuthRg.GET(
		"/:id/logo",
		r.playerThemesController.GetPlayerThemeLogo,
	)

	playerRoute.Use(
		middlewares.NewRateLimiter(
			"/players"),
	)
	playerRoute.Use(r.apiKeyAuth)

	playerRoute.POST(
		"",
		r.playerThemesController.CreatePlayerTheme,
	)
	playerRoute.POST(
		"/:id/logo",
		r.playerThemesController.UploadLogoById,
	)

	playerRoute.POST(
		"/add-player",
		r.playerThemesController.AddPlayerThemesToMedia,
	)

	playerRoute.POST(
		"/remove-player",
		r.playerThemesController.RemovePlayerThemesFromMedia,
	)
	playerRoute.GET(
		"",
		r.playerThemesController.ListAllPlayersThemes,
	)
	playerRoute.GET(
		"/:id",
		r.playerThemesController.RetrievePlayerThemeById,
	)

	playerRoute.PATCH(
		"/:id",
		r.playerThemesController.UpdatePlayerThemeById,
	)
	playerRoute.DELETE(
		"/:id",
		r.playerThemesController.DeletePlayerThemeById,
	)
	playerRoute.DELETE(
		"/:id/logo",
		r.playerThemesController.DeleteLogoById,
	)

}
