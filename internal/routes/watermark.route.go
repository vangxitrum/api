package routes

import (
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/controllers"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/middlewares"
	"github.com/labstack/echo/v4"
)

type WatermarkRoute struct {
	watermarkController *controllers.WatermarkController
	apiKeyAuth          echo.MiddlewareFunc
}

func NewWatermarkRoute(
	watermarkController *controllers.WatermarkController,
	apiKeyAuth echo.MiddlewareFunc,
) *WatermarkRoute {
	return &WatermarkRoute{
		watermarkController: watermarkController,
		apiKeyAuth:          apiKeyAuth,
	}
}

func (r *WatermarkRoute) Register(rg *echo.Group) {
	watermarkRoute := rg.Group("/watermarks")

	watermarkRoute.Use(
		middlewares.NewRateLimiter(
			"/watermarks"),
	)
	watermarkRoute.Use(r.apiKeyAuth)

	watermarkRoute.GET(
		"",
		r.watermarkController.ListAllWaterMarks,
	)
	watermarkRoute.POST(
		"",
		r.watermarkController.CreateWaterMark,
	)
	watermarkRoute.DELETE(
		"/:id",
		r.watermarkController.DeleteWatermarkById,
	)
}
