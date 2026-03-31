package routes

import (
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/controllers"
)

type UserRoute struct {
	userController   *controllers.UserController
	deserializerUser echo.MiddlewareFunc
	apiKeyMiddleware echo.MiddlewareFunc
}

func NewUserRoute(
	userController *controllers.UserController,
	deserializerUser echo.MiddlewareFunc,
	apiKeyMiddleware echo.MiddlewareFunc,
) *UserRoute {
	return &UserRoute{
		userController:   userController,
		deserializerUser: deserializerUser,
		apiKeyMiddleware: apiKeyMiddleware,
	}
}

func (r *UserRoute) Register(rg *echo.Group) {
	rg.GET("/user/subscribe", r.userController.SubscribeForNew)

	userRg := rg.Group("/user")
	authRg := userRg.Group("")
	authRg.Use(r.apiKeyMiddleware)

	authRg.GET("/me", r.userController.GetMe)
	authRg.GET("/code/:code", r.userController.UseExclusiveCode)

	deleteUserRg := userRg.Group("")
	deleteUserRg.Use(r.deserializerUser)
	deleteUserRg.DELETE("", r.userController.DeleteUser)

	authRg.PATCH(
		"/rename",
		r.userController.ChangeUserName,
	)

	authRg.GET(
		"/challenge/:address",
		r.userController.GetUserChallenge,
	)

	authRg.PUT(
		"/link_wallet",
		r.userController.LinkWallet,
	)

	userRg.POST(
		"/join-exclusive-program",
		r.userController.RequestJoinExclusiveProgram,
	)

	authRg.GET("/languages", r.userController.GetSupportLanguages)
}
