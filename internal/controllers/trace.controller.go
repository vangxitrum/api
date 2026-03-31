package controllers

import (
	"bytes"
	"net/http"

	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
)

type TraceController struct{}

func NewTraceController() *TraceController {
	return &TraceController{}
}

func (c *TraceController) GetTraceData(ctx echo.Context) error {
	if response.TraceHelperInstance == nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusInternalServerError,
			"trace helper is nil",
		)
	}

	data, err := response.TraceHelperInstance.Load(ctx.Param("id"))
	if err != nil {
		return response.NewHttpError(http.StatusInternalServerError, err)
	}

	return ctx.Stream(http.StatusOK, echo.MIMEApplicationJSON, bytes.NewReader(data))
}
