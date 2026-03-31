package routes

import (
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/controllers"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/middlewares"
	"github.com/labstack/echo/v4"
)

type WebhookRoute struct {
	webhookController *controllers.WebhookController
	apiKeyAuth        echo.MiddlewareFunc
}

func NewWebhookRoute(
	webhookController *controllers.WebhookController,
	apiKeyAuth echo.MiddlewareFunc,
) *WebhookRoute {
	return &WebhookRoute{
		webhookController: webhookController,
		apiKeyAuth:        apiKeyAuth,
	}
}

func (r *WebhookRoute) Register(rg *echo.Group) {
	webhookRoute := rg.Group("/webhooks")

	webhookRoute.Use(
		middlewares.NewRateLimiter(
			"/webhooks"),
	)
	webhookRoute.Use(r.apiKeyAuth)

	webhookRoute.GET(
		"/:id",
		r.webhookController.GetUserWebhook,
	)
	webhookRoute.POST(
		"",
		r.webhookController.CreateWebhook,
	)
	webhookRoute.GET(
		"",
		r.webhookController.GetWebhookList,
	)
	webhookRoute.PATCH(
		"/:id",
		r.webhookController.UpdateWebhook,
	)
	webhookRoute.DELETE(
		"/:id",
		r.webhookController.DeleteWebhook,
	)

	webhookRoute.POST(
		"/check/:id",
		r.webhookController.CheckWebhookById,
	)
}
