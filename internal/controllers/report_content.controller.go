package controllers

import (
	"net/http"
	"time"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	utils "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/payload"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/validate"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ReportContentController struct {
	reportContentService *services.ReportContentService
}

func NewReportContentController(
	reportContentService *services.ReportContentService,
) *ReportContentController {
	return &ReportContentController{
		reportContentService: reportContentService,
	}
}

type CreateReportContentRequest struct {
	MediaId     uuid.UUID `json:"media_id"`
	MediaType   string    `json:"media_type"`
	Description string    `json:"description"`
	Reason      string    `json:"reason"`
}

// CreateReportContent godoc
//
//	@Summary		Create a report content
//	@Description	Create a report content.
//	@Tags			reports
//	@Accept			json
//	@Produce		json
//	@Param			input	body		CreateReportContentRequest	true	"Create report content request"
//	@Success		201		{object}	models.ResponseSuccess
//	@Failure		400		{object}	models.ResponseError
//	@Failure		403		{object}	models.ResponseError
//	@Failure		500		{object}	models.ResponseError
//	@Router			/reports [post]
func (c *ReportContentController) CreateReportContent(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CreateReportContent").Observe(time.Since(t).Seconds())
	}()

	var payload CreateReportContentRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	mediaId, err := uuid.Parse(payload.MediaId.String())
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid media id.")
	}

	if payload.MediaId == uuid.Nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Media id is required.")
	}

	if len(payload.Description) > models.MaxDescriptionLength {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Description is too long. Max length is 1000 characters.")
	}

	if payload.Reason == "" {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Reason is required.")
	}

	if !models.ValidContentReportReason[models.ContentReportReason(payload.Reason)] {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid reason.")
	}

	if !models.ValidMediaTypes[payload.MediaType] {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid media type. Allowed values: media, live_stream.")
	}

	ReporterIp := ctx.RealIP()

	if err := c.reportContentService.CreateReportContent(
		ctx.Request().Context(),
		mediaId,
		ReporterIp,
		payload.MediaType,
		payload.Description,
		models.ContentReportReason(payload.Reason),
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusCreated, "Report content created successfully.")
}

type GetReportContentListRequest struct {
	SortBy  string `json:"sort_by" form:"sort_by" query:"sort_by"`
	OrderBy string `json:"order_by" form:"order_by" query:"order_by"`
	Offset  uint64 `json:"offset" form:"offset" query:"offset"`
	Limit   uint64 `json:"limit" form:"limit" query:"limit"`
	Status  string `json:"status" form:"status" query:"status"`
}

type GetReportContentListData struct {
	ReportContents []*models.ContentReport `json:"report_contents"`
	Total          int64                   `json:"total"`
}

type GetReportContentListResponse struct {
	Status string                   `json:"status"`
	Data   GetReportContentListData `json:"data"`
}

// GetReportContentList godoc
//
//	@Summary		Get a list of report contents
//	@Description	Get a list of report contents.
//	@Tags			reports
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Security		BasicAuth
//	@Param			input	query		GetReportContentListRequest	true	"Get report content list request"
//	@Success		200		{object}	GetReportContentListResponse
//	@Failure		400		{object}	models.ResponseError
//	@Failure		403		{object}	models.ResponseError
//	@Failure		500		{object}	models.ResponseError
//	@Router			/reports [get]
func (c *ReportContentController) GetReportContentList(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetReportContentList").Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	if !validate.IsValidAdminEmail(authInfo.User.Email) {
		return response.ResponseFailMessage(ctx, http.StatusForbidden, "You are not allowed to access this resource.")
	}

	var payload GetReportContentListRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}
	if payload.SortBy != "" && !models.SortByMap[payload.SortBy] {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Allowed values: created_at.")
	}

	if payload.OrderBy != "" && !models.OrderMap[payload.OrderBy] {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Unknown sorting order. Please use \"asc\" or \"desc\".")
	}

	if payload.Limit > models.MaxPageLimit {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Limit only allowed values 1-100.")
	}

	if payload.Status != "" && !models.ValidContentReportStatus[payload.Status] {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Allowed values: new, resolved.")
	}

	filterPayload := &utils.FilterPayload{
		Limit:  payload.Limit,
		SortBy: payload.SortBy,
		Order:  payload.OrderBy,
	}
	utils.SetDefaultsFilter(filterPayload, 25, "created_at", "desc")

	filter := models.GetContentReportList{
		SortBy: filterPayload.SortBy,
		Order:  filterPayload.Order,
		Status: payload.Status,
		Offset: payload.Offset,
		Limit:  filterPayload.Limit,
	}
	result, total, err := c.reportContentService.GetReportContentList(
		ctx.Request().Context(),
		filter,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		GetReportContentListData{
			ReportContents: result,
			Total:          total,
		})
}

type UpdateReportContentRequest struct {
	Status string `json:"status" form:"status"`
}

// UpdateReportContent godoc
//
//	@Summary		Update a report content
//	@Description	Update a report content.
//	@Tags			reports
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Security		BasicAuth
//	@Param			input	query		UpdateReportContentRequest	true	"Update report content request"
//	@Success		200		{object}	models.ResponseSuccess
//	@Failure		400		{object}	models.ResponseError
//	@Failure		403		{object}	models.ResponseError
//	@Failure		500		{object}	models.ResponseError
//	@Router			/reports/{id} [patch]
func (c *ReportContentController) UpdateReportContent(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("UpdateReportContent").Observe(time.Since(t).Seconds())
	}()

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	if !validate.IsValidAdminEmail(authInfo.User.Email) {
		return response.ResponseFailMessage(ctx, http.StatusForbidden, "You are not allowed to access this resource.")
	}
	var payload UpdateReportContentRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	reportId, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid report id.")
	}

	if payload.Status == "" {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Status is required.")
	}

	if !models.ValidContentReportStatus[payload.Status] {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid status. Allowed values: new, resolved.")
	}

	if err := c.reportContentService.UpdateReportContent(
		ctx.Request().Context(),
		reportId,
		payload.Status,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Report content updated successfully.",
	)
}

// DeleteReportContent godoc
//
//	@Summary		Delete a report content
//	@Description	Delete a report content.
//	@Tags			reports
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Security		BasicAuth
//	@Param			id	path		string	true	"Report ID"
//	@Success		200	{object}	models.ResponseSuccess
//	@Failure		400	{object}	models.ResponseError
//	@Failure		403	{object}	models.ResponseError
//	@Failure		500	{object}	models.ResponseError
//	@Router			/reports/{id} [delete]
func (c *ReportContentController) DeleteReportContent(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("DeleteReportContent").Observe(time.Since(t).Seconds())
	}()
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid report id.")
	}
	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	if !validate.IsValidAdminEmail(authInfo.User.Email) {
		return response.ResponseFailMessage(ctx, http.StatusForbidden, "You are not allowed to access this resource.")
	}

	if err := c.reportContentService.DeleteReportContent(
		ctx.Request().Context(),
		id,
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		"Report content deleted successfully.",
	)
}
