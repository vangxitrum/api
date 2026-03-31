package routes

import (
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/controllers"
)

type UsageRoute struct {
	usageController     *controllers.UsageController
	statisticController *controllers.StatisticController

	deserializerUser echo.MiddlewareFunc
	apiKeyMiddleware echo.MiddlewareFunc
}

func NewUsageRoute(
	usageController *controllers.UsageController,
	statisticController *controllers.StatisticController,

	deserializerUsage echo.MiddlewareFunc,
	apiKeyMiddleware echo.MiddlewareFunc,
) *UsageRoute {
	return &UsageRoute{
		usageController:     usageController,
		statisticController: statisticController,

		deserializerUser: deserializerUsage,
		apiKeyMiddleware: apiKeyMiddleware,
	}
}

func (r *UsageRoute) Register(rg *echo.Group) {
	usageRg := rg.Group("/payment")
	usageRg.Use(r.deserializerUser)

	usageRg.GET("/top_ups", r.usageController.GetUserTopUps)
	usageRg.GET("/billings", r.usageController.GetUserBillings)
	usageRg.GET("/usage", r.usageController.GetUserUsageByInterval)
	usageRg.GET("/convert", r.usageController.ConvertUsdToAioz)

	statisticRg := rg.Group("/statistic")
	statisticRg.POST("/watch_info", r.statisticController.CreateWatchInfo)
	statisticRg.POST("/action", r.statisticController.CreateAction)

	analyticRoute := rg.Group("/analytics")
	analyticRoute.Use(r.apiKeyMiddleware)
	analyticRoute.POST(
		"/metrics/data/:metric/:aggregation",
		r.statisticController.GetAggregatedMetrics,
	)
	analyticRoute.POST(
		"/metrics/bucket/:metric/:breakdown",
		r.statisticController.GetBreakdownMetrics,
	)
	analyticRoute.POST(
		"/metrics/timeseries/:metric/:interval",
		r.statisticController.GetOvertimeMetrics,
	)

	analyticRoute.GET(
		"/media",
		r.statisticController.GetStatisticMedias,
	)

	analyticRoute.GET(
		"/data",
		r.statisticController.GetDataUsage,
	)

	adminMetricsRoute := rg.Group("/metrics")
	adminMetricsRoute.GET("/summary", r.statisticController.GetAdminStatistic)
	adminMetricsRoute.GET("/media", r.statisticController.GetMostViewedMedia)
}
