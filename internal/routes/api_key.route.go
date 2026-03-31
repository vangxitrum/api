package routes

import (
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/controllers"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/middlewares"
	"github.com/labstack/echo/v4"
)

type ApiKeyRoute struct {
	apiKeyController *controllers.ApiKeyController
	apiKeyAuth       echo.MiddlewareFunc
}

func NewApiKeyRoute(
	apiKeyController *controllers.ApiKeyController,
	apiKeyAuth echo.MiddlewareFunc,
) *ApiKeyRoute {
	return &ApiKeyRoute{
		apiKeyController: apiKeyController,
		apiKeyAuth:       apiKeyAuth,
	}
}

func (r *ApiKeyRoute) Register(rg *echo.Group) {
	apiKeyRoute := rg.Group("/api_keys")

	apiKeyRoute.Use(
		middlewares.NewRateLimiter(
			"/api_keys"),
	)
	apiKeyRoute.Use(r.apiKeyAuth)

	apiKeyRoute.POST(
		"",
		r.apiKeyController.CreateApiKey,
	)

	apiKeyRoute.GET(
		"",
		r.apiKeyController.GetApiKeyList,
	)

	apiKeyRoute.PATCH(
		"/:id",
		r.apiKeyController.UpdateApiKey,
	)
	apiKeyRoute.DELETE(
		"/:id",
		r.apiKeyController.DeleteApiKey,
	)
}
