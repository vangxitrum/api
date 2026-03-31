package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

type UsageController struct {
	usageService *services.UsageService
}

func NewUsageController(usageService *services.UsageService) *UsageController {
	return &UsageController{
		usageService: usageService,
	}
}

type GetPaymentRequest struct {
	Offset  int    `json:"offset"  form:"offset"   query:"offset"`
	Limit   int    `json:"limit"   form:"limit"    query:"limit"`
	OrderBy string `json:"orderBy" form:"order_by" query:"order_by"`
	From    int64  `json:"from"    form:"from"     query:"from"`
	To      int64  `json:"to"      form:"to"       query:"to"`
}

type GetTopUpsResponse struct {
	Total  int64           `json:"total"`
	TopUps []*models.TopUp `json:"top_ups"`
}

func (c *UsageController) GetUserTopUps(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetUserTopUps").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetPaymentRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	if payload.From < 0 || payload.To < 0 {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid time range.")
	}

	var from, to time.Time
	from = time.Unix(payload.From, 0)
	if payload.To != 0 {
		to = time.Unix(payload.To, 0)
	} else {
		to = time.Now().UTC()
	}

	if payload.Offset < 0 || payload.Limit < 0 {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid offset or limit.")
	}

	if payload.Limit == 0 {
		payload.Limit = models.PageSizeLimit
	}

	if payload.OrderBy == "" {
		payload.OrderBy = "desc"
	}

	if payload.OrderBy != "asc" && payload.OrderBy != "desc" {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid order by.")
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	rs, total, err := c.usageService.GetUserTopUps(ctx.Request().Context(), models.GetPaymentInput{
		Offset:  payload.Offset,
		Limit:   payload.Limit,
		OrderBy: payload.OrderBy,
		From:    from,
		To:      to,
	}, authInfo)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, GetTopUpsResponse{
		Total:  total,
		TopUps: rs,
	})
}

type GetUserBillingsResponse struct {
	Total    int64             `json:"total"`
	Billings []*models.Billing `json:"billings"`
}

func (c *UsageController) GetUserBillings(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetUserBillings").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetPaymentRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	if payload.From < 0 || payload.To < 0 {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid time range.")
	}

	var from, to time.Time
	from = time.Unix(payload.From, 0)
	if payload.To != 0 {
		to = time.Unix(payload.To, 0)
	} else {
		to = time.Now().UTC()
	}

	if payload.Offset < 0 || payload.Limit < 0 {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid offset or limit.")
	}

	if payload.Limit == 0 {
		payload.Limit = models.PageSizeLimit
	}

	if payload.OrderBy == "" {
		payload.OrderBy = "desc"
	}

	if payload.OrderBy != "asc" && payload.OrderBy != "desc" {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid order by.")
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	rs, total, err := c.usageService.GetUserBillings(
		ctx.Request().Context(),
		models.GetPaymentInput{
			Offset:  payload.Offset,
			Limit:   payload.Limit,
			OrderBy: payload.OrderBy,
			From:    from,
			To:      to,
		},
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, GetUserBillingsResponse{
		Total:    total,
		Billings: rs,
	})
}

type GetUserUsageByIntervalRequest struct {
	From int64 `json:"from" form:"from" query:"from"`
	To   int64 `json:"to"   form:"to"   query:"to"`
}

func (c *UsageController) GetUserUsageByInterval(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetUserUsageByInterval").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetUserUsageByIntervalRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	from := time.Unix(payload.From, 0)
	to := time.Unix(payload.To, 0)
	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	rs, err := c.usageService.GetUserUsageByInterval(ctx.Request().Context(), from, to, authInfo)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, rs)
}

type ConvertUsdToAiozResponse struct {
	Amount float64 `json:"amount"`
}

func (c *UsageController) ConvertUsdToAioz(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("ConvertUsdToAioz").
			Observe(time.Since(t).Seconds())
	}()

	usd, err := strconv.ParseFloat(ctx.QueryParam("amount"), 64)
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid amount.")
	}

	if usd <= 0 {
		return response.ResponseSuccess(ctx, http.StatusOK, ConvertUsdToAiozResponse{
			Amount: 0,
		})
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	aiozAmount, err := c.usageService.ConvertUsdToAioz(ctx.Request().Context(), usd, authInfo)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, ConvertUsdToAiozResponse{
		Amount: aiozAmount,
	})
}
