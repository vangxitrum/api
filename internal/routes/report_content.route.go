package routes

import (
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/controllers"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/middlewares"
	"github.com/labstack/echo/v4"
)

type ReportContentRoute struct {
	reportContentController *controllers.ReportContentController
	apiKeyAuth              echo.MiddlewareFunc
}

func NewReportContentRoute(
	reportContentController *controllers.ReportContentController,
	apiKeyAuth echo.MiddlewareFunc,
) *ReportContentRoute {
	return &ReportContentRoute{
		reportContentController: reportContentController,
		apiKeyAuth:              apiKeyAuth,
	}
}

func (r *ReportContentRoute) Register(rg *echo.Group) {
	reportContentRoute := rg.Group("/reports")

	authRg := reportContentRoute.Group("")
	authRg.Use(r.apiKeyAuth)

	authRg.GET(
		"",
		r.reportContentController.GetReportContentList,
	)

	authRg.PATCH(
		"/:id",
		r.reportContentController.UpdateReportContent,
	)
	authRg.DELETE(
		"/:id",
		r.reportContentController.DeleteReportContent,
	)

	noAuthRg := reportContentRoute.Group("")
	noAuthRg.Use(
		middlewares.NewRateLimiter(
			"/reports"),
	)
	noAuthRg.POST("", r.reportContentController.CreateReportContent)
}
