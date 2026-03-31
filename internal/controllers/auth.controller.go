package controllers

import (
	"net/http"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/labstack/echo/v4"

	_ "10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	mails "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/mail"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

type AuthController struct {
	userService *services.UserService
}

func NewAuthController(userService *services.UserService) *AuthController {
	return &AuthController{
		userService: userService,
	}
}

type LoginCodeRequest struct {
	Email string `json:"email" form:"email"`
	Code  string `json:"code"  form:"code"`
} //	@name	LoginCodeRequest

type LoginCodeData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
} //	@name	LoginCodeData

type LoginCodeResponse struct {
	Status string        `json:"status"`
	Data   LoginCodeData `json:"data"`
} //	@name	LoginCodeResponse

// LoginCode godoc
//
//	@Summary		Login code
//	@Description	Login with code sent to email
//	@Tags			auth
//	@Accept			json
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Param			data	body		LoginCodeRequest	true	"input"
//	@Success		200		{object}	LoginCodeResponse
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
//	@Router			/auth/login_code [post]
func (c *AuthController) LoginCode(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("LoginWithCode").
			Observe(time.Since(t).Seconds())
	}()
	var payload LoginCodeRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	formattedEmail, err := mails.FormatEmail(payload.Email)
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid email.")
	}

	accessToken, refreshToken, err := c.userService.VerifyCode(
		ctx.Request().Context(),
		formattedEmail,
		payload.Code,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		LoginCodeData{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	)
}

type EmailExistedData struct {
	Email     string `json:"email"`
	ExpiredAt int64  `json:"expired_at"`
} //	@name	EmailExistedData
type EmailExistedResponse struct {
	Status string           `json:"status"`
	Data   EmailExistedData `json:"data"`
} //	@name	EmailExistedResponse

// SendMailLogin godoc
//
//	@Summary		Send mail login
//	@Description	Send code to login. If user already login within 1 minute, return expired time
//	@Tags			auth
//	@Accept			json
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Param			email	path		string					true	"email"
//	@Success		200		{object}	EmailExistedResponse	"success"
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
//	@Router			/auth/email_login/{email} [get]
func (c *AuthController) SendMailLogin(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetLoginCode").Observe(time.Since(t).Seconds())
	}()

	email, err := url.QueryUnescape(ctx.Param("email"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid email.")
	}

	formattedEmail, err := mails.FormatEmail(email)
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid email.")
	}

	expiredAt, err := c.userService.SendLoginCode(
		ctx.Request().Context(),
		formattedEmail,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		EmailExistedData{
			Email:     email,
			ExpiredAt: expiredAt,
		},
	)
}

type EmailSignUpData struct {
	Email     string `json:"email"`
	ExpiredAt int64  `json:"expired_at"`
} //	@name	EmailSignUpData

type EmailSignUpResponse struct {
	Status string          `json:"status"`
	Data   EmailSignUpData `json:"data"`
} //	@name	EmailSignUpResponse

// SendMailSignUp godoc
//
//	@Summary		Send mail sign up
//	@Description	Send code to sign up. If user already sign up within 1 minute, return expired time
//	@Tags			auth
//	@Accept			json
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Param			email	path		string					true	"email"
//	@Success		200		{object}	EmailSignUpResponse		"success"
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
//	@Router			/auth/email_signup/{email} [get]
func (c *AuthController) SendMailSignUp(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("SendMailSignUp").Observe(time.Since(t).Seconds())
	}()

	email, err := url.QueryUnescape(ctx.Param("email"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid email.")
	}

	formattedEmail, err := mails.FormatEmail(email)
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid email.")
	}

	expiredAt, err := c.userService.SendSignUpCode(
		ctx.Request().Context(),
		formattedEmail,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		EmailExistedData{
			Email:     email,
			ExpiredAt: expiredAt,
		},
	)
}

type RefreshTokenData struct {
	AccessToken string `json:"access_token"`
} //	@name	RefreshTokenData

type RefreshTokenResponse struct {
	Status string           `json:"status"`
	Data   RefreshTokenData `json:"data"`
} //	@name	RefreshTokenResponse

// RefreshAccessToken godoc
//
//	@Summary		Refresh access token
//	@Description	Refresh access token using refresh token cookie
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			refresh_token	header		string	true	"Refresh Token"
//	@Success		200				{object}	RefreshTokenResponse
//	@Header			200				{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			200				{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			200				{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		400				{object}	models.ResponseError
//	@Header			400				{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			400				{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			400				{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		403				{object}	models.ResponseError
//	@Header			403				{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			403				{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			403				{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		404				{object}	models.ResponseError
//	@Header			404				{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			404				{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			404				{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure		500				{object}	models.ResponseError
//	@Header			500				{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header			500				{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header			500				{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router			/auth/refresh [get]
func (c *AuthController) RefreshAccessToken(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("RefreshToken").Observe(time.Since(t).Seconds())
	}()

	refreshToken := ctx.Request().Header.Get("refresh_token")
	if refreshToken == "" {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Refresh token is required.")
	}

	accessToken, err := c.userService.RefreshToken(
		ctx.Request().Context(),
		refreshToken,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		RefreshTokenData{AccessToken: accessToken},
	)
}

// GetChallenge godoc
//
//	@Summary		Challenge
//	@Description	Return challenge for metamask signature
//	@Tags			auth
//	@Accept			json
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Param			address	path		string					true	"address"
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
//	@Router			/auth/challenge/{address} [get]
func (c *AuthController) GetChallenge(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetWalletChallenge").
			Observe(time.Since(t).Seconds())
	}()

	isValidAddress := common.IsHexAddress(ctx.Param("address"))
	if !isValidAddress {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid address.")

	}

	address := common.HexToAddress(ctx.Param("address"))

	challenge, err := c.userService.GetChallenge(
		ctx.Request().Context(),
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

type AuthorizeRequest struct {
	Address   string `json:"address"   form:"address"   binding:"required"`
	Signature string `json:"signature" form:"signature" binding:"required"`
} //	@name	AuthorizeRequest

type AuthorizeData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
} //	@name	AuthorizeData

type AuthorizeResponse struct {
	Status string        `json:"status"`
	Data   AuthorizeData `json:"data"`
} //	@name	AuthorizeResponse

// Authorize godoc
//
//	@Summary		Authorize
//	@Description	Authorize metamask signature
//	@Tags			auth
//	@Accept			json
//	@Accept			x-www-form-urlencoded
//	@Produce		json
//	@Param			AuthorizeRequest	body		AuthorizeRequest		true	"address, signature"
//	@Success		200					{object}	AuthorizeResponse		"success"
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
//	@Router			/auth/authorize [post]
func (c *AuthController) Authorize(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("AuthorizeWalletChallenge").
			Observe(time.Since(t).Seconds())
	}()

	var r AuthorizeRequest
	if err := ctx.Bind(&r); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	isValidAddress := common.IsHexAddress(r.Address)
	if !isValidAddress {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid address.")
	}

	address := common.HexToAddress(r.Address)
	accessToken, refreshToken, err := c.userService.VerifyChallenge(
		ctx.Request().Context(),
		address,
		r.Signature,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}
	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		AuthorizeData{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	)
}
