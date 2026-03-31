package routes

import (
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/controllers"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/middlewares"
)

type LiveStreamRoute struct {
	liveStreamController *controllers.LiveStreamController
	apiKeyAuth           echo.MiddlewareFunc
}

func NewLiveStreamRoute(
	liveStreamController *controllers.LiveStreamController,
	apiKeyAuth echo.MiddlewareFunc,
) *LiveStreamRoute {
	return &LiveStreamRoute{
		liveStreamController: liveStreamController,
		apiKeyAuth:           apiKeyAuth,
	}
}

func (r *LiveStreamRoute) Register(rg *echo.Group) {
	liveStreamRg := rg.Group("/live_streams")
	liveStreamRg.Use(r.apiKeyAuth)
	liveStreamRg.Use(
		middlewares.NewRateLimiter(
			"/live_streams"),
	)
	{
		liveStreamRg.GET("", r.liveStreamController.GetLiveStreamKeys)
		liveStreamRg.GET("/:id", r.liveStreamController.GetLiveStreamKey)
		liveStreamRg.GET("/:id/streamings", r.liveStreamController.GetStreamings)
		liveStreamRg.GET("/:id/streamings/:stream_id", r.liveStreamController.GetStreaming)
		liveStreamRg.GET("/:id/video", r.liveStreamController.GetLiveStreamMedia)
		liveStreamRg.GET(
			"/multicast/:stream_key",
			r.liveStreamController.GetLiveStreamMulticastByStreamKey,
		)
		liveStreamRg.GET(
			"/statistic/:stream_media_id",
			r.liveStreamController.GetLiveStreamStatisticByStreamMediaId,
		)

		liveStreamRg.POST("", r.liveStreamController.CreateLiveStreamKey)
		liveStreamRg.POST("/:id/media", r.liveStreamController.GetLiveStreamMedias)
		liveStreamRg.POST("/:id/streamings", r.liveStreamController.CreateStreaming)
		liveStreamRg.POST("/multicast/:stream_key", r.liveStreamController.AddLiveStreamMulticast)
		liveStreamRg.POST("/:id/thumbnail", r.liveStreamController.UploadLiveStreamThumbnail)

		liveStreamRg.DELETE("/:id/thumbnail", r.liveStreamController.DeleteLiveStreamThumbnail)
		liveStreamRg.DELETE("/:id/streamings/:stream_id", r.liveStreamController.DeleteStreaming)
		liveStreamRg.DELETE("/:id", r.liveStreamController.DeleteLiveStreamKey)
		liveStreamRg.DELETE(
			"/multicast/:stream_key",
			r.liveStreamController.DeleteLiveStreamMulticast,
		)

		liveStreamRg.PUT("/:id/streamings", r.liveStreamController.UpdateLiveStreamMedia)
		liveStreamRg.PUT("/:id", r.liveStreamController.UpdateLiveStreamKey)
	}

	publicLiveStreamRg := rg.Group("/live_streams")
	{
		publicLiveStreamRg.GET(
			"/player/:id/media",
			r.liveStreamController.GetLiveStreamMediaPublic,
		)
		publicLiveStreamRg.GET(
			"/webhook/connect",
			r.liveStreamController.GetConnectLiveStreamWebhook,
		)
		publicLiveStreamRg.GET(
			"/webhook/disconnect",
			r.liveStreamController.GetDisconnectLiveStreamWebhook,
		)
	}
}
