package routes

import (
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/controllers"
)

type TraceRoute struct {
	traceController *controllers.TraceController
}

func NewTraceRoute(
	traceController *controllers.TraceController,
) *TraceRoute {
	return &TraceRoute{
		traceController: traceController,
	}
}

func (r *TraceRoute) Register(rg *echo.Group) {
	traceRoute := rg.Group("/trace")

	traceRoute.GET("/:id", r.traceController.GetTraceData)
}
