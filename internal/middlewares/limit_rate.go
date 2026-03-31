package middlewares

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"github.com/labstack/echo/v4"
	"github.com/patrickmn/go-cache"
)

type RateLimitInfo struct {
	Limit     int
	Remaining int
	ResetTime time.Time
}

var limiterSet = cache.New(5*time.Minute, 10*time.Minute)
var mediaPartRegex = regexp.MustCompile(`media/.+/part`)
var mediaApiRegex = regexp.MustCompile(`media(/([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}))?$`)

func NewRateLimiter(path string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			duration := models.RetryAfterDuration
			key := ""
			fullPath := ctx.Request().URL.Path

			if mediaApiRegex.MatchString(fullPath) {
				key = ctx.RealIP() + path + "_"
			} else if mediaPartRegex.MatchString(fullPath) {
				key = ctx.RealIP() + fullPath + "_" + ctx.Request().Method
			} else {
				key = ctx.RealIP() + path + "_" + ctx.Request().Method
			}

			var info *RateLimitInfo
			if data, found := limiterSet.Get(key); found {
				var ok bool
				if info, ok = data.(*RateLimitInfo); !ok {
					slog.ErrorContext(ctx.Request().Context(), "RateLimitInfo type assertion failed.")
				}
			} else {
				limit := models.WritesRateLimit
				if ctx.Request().Method == http.MethodGet {
					limit = models.ReadsRateLimit
				} else if ctx.Request().Method == http.MethodPost && mediaPartRegex.MatchString(fullPath) {
					limit = models.UploadRateLimit
				} else if mediaApiRegex.MatchString(fullPath) {
					limit = models.ReadsRateLimit
				}
				info = &RateLimitInfo{
					Limit:     limit,
					Remaining: limit,
					ResetTime: time.Now().Add(duration),
				}
				limiterSet.Set(key, info, duration)
			}

			now := time.Now()
			if now.After(info.ResetTime) {
				info.Remaining = info.Limit
				info.ResetTime = now.Add(duration)
			}

			retryAfter := int(info.ResetTime.Sub(now).Seconds())
			if retryAfter < 0 {
				retryAfter = 0
			}

			ctx.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
			ctx.Response().Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))
			ctx.Response().Header().Set("X-RateLimit-Retry-After", fmt.Sprintf("%d", retryAfter))

			if info.Remaining == 0 {
				return response.ResponseFailMessage(ctx, http.StatusTooManyRequests, fmt.Sprintf("Too many requests. Please wait %d seconds before retrying.", retryAfter))
			}

			info.Remaining--
			limiterSet.Set(key, info, duration)

			return next(ctx)
		}
	}
}
