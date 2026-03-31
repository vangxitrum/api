package routes

import (
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/controllers"
)

type AuthRoute struct {
	authController *controllers.AuthController
}

func NewAuthRoute(
	authController *controllers.AuthController,
) *AuthRoute {
	return &AuthRoute{
		authController: authController,
	}
}

func (r *AuthRoute) Register(rg *echo.Group) {
	authRoute := rg.Group("/auth")

	// add signup mail
	authRoute.POST("/authorize", r.authController.Authorize)
	authRoute.POST("/login_code", r.authController.LoginCode)

	authRoute.GET("/email_login/:email", r.authController.SendMailLogin)
	authRoute.GET("/email_signup/:email", r.authController.SendMailSignUp)
	authRoute.GET("/challenge/:address", r.authController.GetChallenge)
	authRoute.GET("/refresh", r.authController.RefreshAccessToken)
}
