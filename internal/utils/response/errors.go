package response

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

var (
	UnauthorizedError = echo.NewHTTPError(
		http.StatusUnauthorized,
		"You are not log in.",
	)
	CodeExpiredError = echo.NewHTTPError(
		http.StatusUnauthorized,
		"Code expired.",
	)
	InvalidSignatureError = echo.NewHTTPError(
		http.StatusBadRequest,
		"Invalid signature.",
	)
)

func NewInternalServerError(err error) *echo.HTTPError {
	return NewHttpError(
		http.StatusInternalServerError,
		err,
		"Internal server error.",
	)
}

func NewPaymentFailError() *echo.HTTPError {
	return NewHttpError(
		http.StatusPaymentRequired,
		nil,
		"Your last payment was failed. Please try again.",
	)
}

func NewBadRequestError(message string) *echo.HTTPError {
	return NewHttpError(
		http.StatusBadRequest,
		nil,
		message,
	)
}

func NewNotFoundError(err error) *echo.HTTPError {
	return NewHttpError(
		http.StatusNotFound,
		err,
		"Record not found.",
	)
}

func NewBadGatewayError(err error) *echo.HTTPError {
	return NewHttpError(
		http.StatusBadGateway,
		err,
		"Bad Gateway.",
	)
}

func NewUnauthorizedError(err error) *echo.HTTPError {
	return NewHttpError(
		http.StatusUnauthorized,
		err,
		"Unauthorized.",
	)
}
