package routes

import (
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/controllers"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/middlewares"
)

type MediaRoute struct {
	mediaController        *controllers.MediaController
	mediaCaptionController *controllers.MediaCaptionController
	mediaChapterController *controllers.MediaChapterController

	apiKeyAuth echo.MiddlewareFunc
}

func NewMediaRoute(
	mediaController *controllers.MediaController,
	mediaCaptionController *controllers.MediaCaptionController,
	mediaChapterController *controllers.MediaChapterController,

	apiKeyAuth echo.MiddlewareFunc,
) *MediaRoute {
	return &MediaRoute{
		mediaController:        mediaController,
		mediaCaptionController: mediaCaptionController,
		mediaChapterController: mediaChapterController,

		apiKeyAuth: apiKeyAuth,
	}
}

func (r *MediaRoute) Register(rg *echo.Group) {
	mediaRg := rg.Group("/media")

	authRg := mediaRg.Group("")
	authRg.Use(r.apiKeyAuth)
	authRg.Use(
		middlewares.NewRateLimiter(
			"/media"),
	)
	authRg.POST("/:id/part", r.mediaController.UploadPart)

	authRg.GET("/cost", r.mediaController.GetTranscodeCost)
	authRg.GET("/:id", r.mediaController.GetMediaDetail)
	authRg.GET(
		"/:id/complete",
		r.mediaController.UploadMediaComplete,
	)
	authRg.GET("/:id/source", r.mediaController.GetMediaSource)
	authRg.GET("/:id/captions", r.mediaCaptionController.GetMediaCaptions)
	authRg.GET("/:id/chapters", r.mediaChapterController.GetMediaChapters)

	authRg.POST("", r.mediaController.GetMediaList)
	authRg.POST("/create", r.mediaController.CreateMediaObject)
	authRg.POST("/:id/thumbnail", r.mediaController.UploadMediaThumbnail)
	authRg.POST("/:id/captions/:lan", r.mediaCaptionController.CreateMediaCaption)
	authRg.POST("/:id/chapters/:lan", r.mediaChapterController.CreateMediaChapter)

	authRg.PATCH("/:id", r.mediaController.UpdateMediaInfo)
	authRg.PATCH("/:id/captions/:lan", r.mediaCaptionController.SetDefaultCaption)

	authRg.DELETE("/:id", r.mediaController.DeleteMedia)
	authRg.DELETE("/:id/thumbnail", r.mediaController.DeleteMediaThumbnail)
	authRg.DELETE("/:id/captions/:lan", r.mediaCaptionController.DeleteMediaCaption)
	authRg.DELETE("/:id/chapters/:lan", r.mediaChapterController.DeleteMediaChapter)

	noAuthRg := mediaRg.Group("")
	noAuthRg.GET("/:id/manifest.m3u8", r.mediaController.GetMediaHlsManifest)
	noAuthRg.GET("/:id/demo", r.mediaController.GetDemoManifest)
	noAuthRg.GET("/:id/manifest.mpd", r.mediaController.GetMediaDashManifest)
	noAuthRg.GET("/:id/mp4", r.mediaController.GetMediaMp4)
	noAuthRg.GET("/:id/captions/:lan", r.mediaController.GetMediaCaption)
	noAuthRg.GET("/:id/chapters/:lan", r.mediaController.GetMediaChapter)
	noAuthRg.GET("/demo-player.json", r.mediaController.GetDemoMediaObject)
	noAuthRg.GET("/:id/player.json", r.mediaController.GetMediaObject)
	noAuthRg.GET("/:id/audio.mp3", r.mediaController.GetMediaAudio)
	noAuthRg.GET(
		"/:id/thumbnail",
		r.mediaController.GetMediaThumbnail,
	)
	noAuthRg.GET(
		"/vod/:id/:filename",
		r.mediaController.GetMediaContent,
	)
}
