package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	echoSwagger "github.com/swaggo/echo-swagger"

	_ "10.0.0.50/tuan.quang.tran/vms-v2/docs"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/middlewares"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
)

func main() {
	// check health
	server.GET(
		"/ping", func(ctx echo.Context) error {
			return ctx.String(
				http.StatusOK,
				"pong",
			)
		},
	)

	// swagger docs
	server.GET(
		"/swagger/*",
		echoSwagger.WrapHandler,
	)

	// pprof
	go func() {
		if err := http.ListenAndServe(":6060", nil); err != nil {
			slog.Error("pprof server", "err", err)
		}
	}()

	// metrics
	server.GET("/metrics", func(c echo.Context) error {
		promhttp.Handler().ServeHTTP(c.Response().Writer, c.Request())
		return nil
	})

	// request size limit
	server.Use(middleware.BodyLimit("50M"))

	// payment client
	paymentClient.StartWatchingPayment(context.Background())

	// start cron jobs
	cron.Start()

	// recover
	server.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize: 1 << 10, // 1 KB
		LogLevel:  log.ERROR,
	}))

	// cors config
	server.Use(
		middleware.CORSWithConfig(
			middleware.CORSConfig{
				AllowOrigins: []string{"*"},
				AllowHeaders: []string{
					echo.HeaderOrigin,
					echo.HeaderContentType,
					echo.HeaderAccept,
					echo.HeaderAuthorization,
					"refresh_token",
					"content-range",
					models.StreamPublicKeyHeader,
					models.StreamSecretKeyHeader,
					"Sec-Ch-Ua",
					"Sec-Ch-Ua-Mobile",
					"Sec-Ch-Ua-Platform",
					models.StreamTraceIdHeader,
					models.AdminApiKeyHeader,
				},
				ExposeHeaders: []string{
					"X-RateLimit-Limit",
					"X-RateLimit-Remaining",
					"X-RateLimit-Retry-After",
				},
				AllowMethods: []string{
					http.MethodGet,
					http.MethodPost,
					http.MethodPut,
					http.MethodPatch,
					http.MethodDelete,
					http.MethodOptions,
				},
			},
		),
	)

	// custom not found route
	server.RouteNotFound("/*", func(c echo.Context) error {
		return c.JSON(http.StatusNotFound, map[string]any{
			"message": "Route not found",
			"path":    c.Request().URL.Path,
			"method":  c.Request().Method,
		})
	})

	// service cron
	// mediaService.StartWatchMediaStatus(context.Background())
	// mediaService.StartWatchQualityStatus(context.Background())
	mediaService.StartWatchPlaylist(context.Background())

	// register routes
	apiRg := server.Group("/api")

	// logs middleware
	apiRg.Use(middlewares.AddLogContext())

	authRoute.Register(apiRg)
	mediaRoute.Register(apiRg)
	userRoute.Register(apiRg)
	apiKeyRoute.Register(apiRg)
	webhookRoute.Register(apiRg)
	playerThemeRoute.Register(apiRg)
	watermarkRoute.Register(apiRg)
	usageRoute.Register(apiRg)
	liveStreamRoute.Register(apiRg)
	reportContentRoute.Register(apiRg)
	playlistRoute.Register(apiRg)
	traceRoute.Register(apiRg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		if err := server.Start(fmt.Sprintf(":%s", appConfig.Port)); err != nil &&
			err != http.ErrServerClosed {
			server.Logger.Fatal("shutting down the server", err)
		}
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		server.Logger.Fatal(err)
	}

	if response.TraceHelperInstance != nil {
		response.TraceHelperInstance.Close()
	}

	if logClient != nil {
		logClient.Close()
	}

	paymentClient.StopWatching()
	mediaService.StopCron()

	<-cron.Stop()
}

//	@title			Aioz Stream API
//	@version		1.0
//	@description	Aioz Stream Service
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@BasePath	/api/

//	@externalDocs.description	aiozstream
//	@externalDocs.url			https://swagger.io/resources/open-api/

//	@securityDefinitions.basic	BasicAuth

//	@securityDefinitions.apikey	Bearer
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space and JWT token.
