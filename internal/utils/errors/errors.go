package middlewares

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

var (
	NotFoundError       = echo.NewHTTPError(http.StatusNotFound, "Record not found.")
	InternalServerError = echo.NewHTTPError(
		http.StatusInternalServerError,
		"Internal server error.",
	)
)
