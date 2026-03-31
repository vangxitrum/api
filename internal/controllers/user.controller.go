package controllers

import (
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	mails "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/mail"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

type UserController struct {
	userService *services.UserService
}

func NewUserController(userService *services.UserService) *UserController {
	return &UserController{
		userService: userService,
	}
}

type GetMeData struct {
	User *models.User `json:"user"`
} //	@name	GetUserData

type GetMeResponse struct {
	Status string    `json:"status"`
	Data   GetMeData `json:"data"`
} //	@name	GetMeResponse

// GetMe godoc
//
//	@Summary		Get me
//	@Description	get current user's information
//	@Tags			user
//	@Security		BasicAuth
//	@Security		Bearer
//	@Param			Authorization	header	string	false	"authorization"
//	@Accept			json
//	@Produce		json
//	@Accept			x-www-form-urlencoded
//	@Success		200	{object}	GetMeResponse
//	@Header			200	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			200	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			200	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		400	{object}	models.ResponseError
//	@Header			400	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			400	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			400	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		403	{object}	models.ResponseError
//	@Header			403	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			403	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			403	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		404	{object}	models.ResponseError
//	@Header			404	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			404	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			404	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		500	{object}	models.ResponseError
//	@Header			500	{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			500	{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			500	{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router			/user/me [get]
func (c *UserController) GetMe(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMe").
			Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	if authInfo.User == nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid auth info.")
	}
	authInfo.User.FullName = authInfo.User.FirstName + " " + authInfo.User.LastName
	user, err := c.userService.GetMe(ctx.Request().Context(), authInfo)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		GetMeData{
			User: user,
		},
	)
}

type ChangeUserNameRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
} //	@name	ChangeUserNameRequest

// ChangeUserName godoc
//
//	@Summary	Change full name
//	@Tags		user
//	@Accept		json
//	@Security	BearerAuth
//	@Accept		x-www-form-urlencoded
//	@Produce	json
//	@Param		data	body		ChangeUserNameRequest	true	"Full name"
//	@Success	200		{object}	models.ResponseSuccess
//	@Header		200		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header		200		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header		200		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure	400		{object}	models.ResponseError
//	@Header		400		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header		400		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header		400		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure	403		{object}	models.ResponseError
//	@Header		403		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header		403		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header		403		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure	404		{object}	models.ResponseError
//	@Header		404		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header		404		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header		404		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure	500		{object}	models.ResponseError
//	@Header		500		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header		500		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header		500		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router		/user/rename [patch]
func (c *UserController) ChangeUserName(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("ChangeUserFullName").
			Observe(time.Since(t).Seconds())
	}()
	userId := ctx.Get("authInfo").(models.AuthenticationInfo).User.Id
	var payload ChangeUserNameRequest

	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid parameters.")
	}

	if payload.FirstName == "" && payload.LastName == "" {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"First name and last name are empty.",
		)
	}

	if err := c.userService.ChangeUserName(
		ctx.Request().Context(),
		userId,
		strings.TrimSpace(payload.FirstName),
		strings.TrimSpace(payload.LastName),
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Change full name successfully.",
	)
}

func (c *UserController) UseExclusiveCode(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UseExclusiveCode").
			Observe(time.Since(t).Seconds())
	}()

	authInfo, ok := ctx.Get("authInfo").(models.AuthenticationInfo)
	if !ok {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid auth info.")
	}

	code := ctx.Param("code")
	if code == "" {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Code is required.")
	}

	if err := c.userService.UseExclusiveCode(
		ctx.Request().Context(),
		code,
		authInfo,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Use exclusive code successfully.",
	)
}

// LinkWallet godoc
//
//	@Summary		Link wallet
//	@Description	Link wallet with existing account by signature and address from metamask
//	@Tags			user
//	@Accept			json
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Security		Bearer
//	@Security		BasicAuth
//	@Param			AuthorizeRequest	body		AuthorizeRequest		true	"Link wallet request"
//	@Success		200					{object}	models.ResponseSuccess	"success"
//	@Header			200					{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			200					{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			200					{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		400					{object}	models.ResponseError
//	@Header			400					{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			400					{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			400					{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		403					{object}	models.ResponseError
//	@Header			403					{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			403					{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			403					{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		404					{object}	models.ResponseError
//	@Header			404					{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			404					{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			404					{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		500					{object}	models.ResponseError
//	@Header			500					{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			500					{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			500					{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router			/user/link_wallet [put]
func (c UserController) LinkWallet(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("LinkWallet").
			Observe(time.Since(t).Seconds())
	}()
	var payload AuthorizeRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid parameters.")
	}

	isValidAddress := common.IsHexAddress(payload.Address)
	if !isValidAddress {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid address.")
	}

	address := common.HexToAddress(payload.Address)
	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	if err := c.userService.LinkWallet(
		ctx.Request().Context(),
		authInfo,
		address,
		payload.Signature,
	); err != nil {
		return response.ResponseError(ctx, err)
	}
	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Link wallet successful.",
	)
}

// GetUserChallenge godoc
//
//	@Summary		Get user challenge
//	@Description	Challenge for wallet verify. This will return a challenge string for signing
//	@Tags			user
//	@Accept			json
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Security		Bearer
//	@Security		BasicAuth
//	@Param			address	path		string					true	"Link wallet request"
//	@Success		200		{object}	models.ResponseSuccess	"success"
//	@Header			200		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			200		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			200		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		400		{object}	models.ResponseError
//	@Header			400		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			400		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			400		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		403		{object}	models.ResponseError
//	@Header			403		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			403		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			403		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		404		{object}	models.ResponseError
//	@Header			404		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			404		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			404		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		500		{object}	models.ResponseError
//	@Header			500		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			500		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			500		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router			/user/challenge/{address} [get]
func (c UserController) GetUserChallenge(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetUserChallenge").
			Observe(time.Since(t).Seconds())
	}()
	isValidAddress := common.IsHexAddress(ctx.Param("address"))

	if !isValidAddress {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid address.")
	}
	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	address := common.HexToAddress(ctx.Param("address"))
	challenge, err := c.userService.GetUserChallenge(
		ctx.Request().Context(),
		authInfo.User.Id,
		address,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		challenge,
	)
}

func (c *UserController) SubscribeForNew(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("SubscribeForNew").
			Observe(time.Since(t).Seconds())
	}()

	formattedEmail, err := mails.FormatEmail(ctx.QueryParam("email"))
	if err == nil {
		c.userService.CreateSubscribeInfo(ctx.Request().Context(), formattedEmail)
	}

	return ctx.JSON(http.StatusOK, struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}{
		Message: "Subscribe successfully.",
		Status:  "success",
	})
}

func (c *UserController) GetSupportLanguages(
	ctx echo.Context,
) error {
	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		models.LanToLanguageMapping,
	)
}

type JoinExclusiveProgramRequest struct {
	OrgName              string  `json:"org_name"`
	Role                 string  `json:"role"`
	Content              string  `json:"content"`
	StorageUsage         float64 `json:"storage_usage"`
	DeliveryUsage        float64 `json:"delivery_usage"`
	UsedStreamPlatforms  string  `json:"used_stream_platforms"`
	HeardAboutAIOZStream string  `json:"heard_about_aioz_stream"`
	Email                string  `json:"email"`
}

func (c *UserController) RequestJoinExclusiveProgram(
	ctx echo.Context,
) error {
	var payload JoinExclusiveProgramRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	if len(strings.TrimSpace(payload.OrgName)) == 0 {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Organization name is required.",
		)
	}

	formattedEmail, err := mails.FormatEmail(payload.Email)
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid email.")
	}

	if len(strings.TrimSpace(payload.Role)) == 0 {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Role is required.",
		)
	}

	if len(strings.TrimSpace(payload.Content)) == 0 {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Content is required.",
		)
	}

	if len(strings.TrimSpace(payload.HeardAboutAIOZStream)) == 0 {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Reason is required.",
		)
	}

	if payload.StorageUsage < 0 || payload.DeliveryUsage < 0 {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Usase must be greater than 0",
		)
	}

	if err := c.userService.RequestJoinExclusiveProgram(ctx.Request().Context(), models.NewJoinExclusiveProgramRequest(
		strings.TrimSpace(payload.OrgName),
		formattedEmail,
		strings.TrimSpace(payload.Role),
		strings.TrimSpace(payload.Content),
		math.Ceil(payload.StorageUsage),
		math.Ceil(payload.DeliveryUsage),
		payload.UsedStreamPlatforms,
		strings.TrimSpace(payload.HeardAboutAIOZStream),
	)); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Request join exclusive program successfully.",
	)
}

func (c *UserController) DeleteUser(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteUser").
			Observe(time.Since(t).Seconds())
	}()

	authInfo, ok := ctx.Get("authInfo").(models.AuthenticationInfo)
	if !ok {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid auth info.")
	}

	if err := c.userService.DeleteUser(
		ctx.Request().Context(),
		authInfo.User.Id,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Delete user successfully.",
	)
}
