package controllers

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

type StatisticController struct {
	statisticService *services.StatisticService
}

func NewStatisticController(
	statisticService *services.StatisticService,
) *StatisticController {
	return &StatisticController{
		statisticService: statisticService,
	}
}

type CreateWatchInfoRequest struct {
	SessionId string                  `json:"session_id"`
	MediaId   string                  `json:"media_id"`
	MediaType string                  `json:"media_type"`
	Type      string                  `json:"type"`
	Data      []*models.WatchInfoItem `json:"data"`
}

func (c *StatisticController) CreateWatchInfo(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CreateWatchInfo").
			Observe(time.Since(t).Seconds())
	}()

	var payload CreateWatchInfoRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	sessionId, err := uuid.Parse(payload.SessionId)
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid session id.")
	}

	mediaId, err := uuid.Parse(payload.MediaId)
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid media id.")
	}

	if payload.MediaType != models.VideoMediaType &&
		payload.MediaType != models.StreamMediaType &&
		payload.MediaType != models.AudioMediaType {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid media type.")
	}

	for _, item := range payload.Data {
		if item.MediaWidth < 0 || item.MediaHeight < 0 || item.MediaAt < 0 {
			return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid media info.")
		}
	}

	deviceType := models.GetDeviceType(
		ctx.Request().Header.Get("User-Agent"),
	)
	var secUaBrowser string
	for k := range models.ValidBrowsers {
		if strings.Contains(strings.ToLower(ctx.Request().Header.Get("Sec-Ch-Ua")), k) {
			secUaBrowser = k
			break
		}
	}

	deviceInfo := models.GetDeviceInfo(ctx.Request().Header.Get("User-Agent"))
	if err := c.statisticService.CreateWatchInfo(
		ctx.Request().Context(),
		models.CreateWatchInfoInput{
			SessionId: sessionId,
			MediaId:   mediaId,
			MediaType: payload.MediaType,
			Data:      payload.Data,
			Type:      payload.Type,
		},
		models.SessionInfo{
			SessionId:      sessionId,
			IP:             ctx.RealIP(),
			DeviceType:     deviceType,
			OperatorSystem: deviceInfo.OperatorSystem,
			Browser:        deviceInfo.Browser,
			UserAgent:      ctx.Request().Header.Get("User-Agent"),
			SecUaBrowser:   secUaBrowser,
		},
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, "Create watch info successfully.")
}

type CreateActionRequest struct {
	SessionId string               `json:"session_id" form:"session_id"`
	MediaId   string               `json:"media_id"   form:"media_id"`
	MediaType string               `json:"media_type" form:"media_type"`
	Data      []*models.ActionItem `json:"data"`
}

func (c *StatisticController) CreateAction(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("CreateAction").
			Observe(time.Since(t).Seconds())
	}()

	var payload CreateActionRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	sessionId, err := uuid.Parse(payload.SessionId)
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid session id.")
	}

	mediaId, err := uuid.Parse(payload.MediaId)
	if err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid media id.")
	}

	if payload.MediaType != models.VideoMediaType &&
		payload.MediaType != models.StreamMediaType &&
		payload.MediaType != models.AudioMediaType {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid media type.")
	}

	for _, item := range payload.Data {
		if item.MediaAt < 0 {
			return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid media info.")
		}
	}

	// var browser string = "unknown"
	// for k := range models.ValidBrowsers {
	// 	data := strings.Split(strings.ToLower(ctx.Request().Header.Get("User-Agent")), " ")
	// 	if strings.Contains(data[len(data)-1], k) {
	// 		browser = k
	// 		break
	// 	}
	// }
	//
	var secUaBrowser string
	for k := range models.ValidBrowsers {
		if strings.Contains(strings.ToLower(ctx.Request().Header.Get("Sec-Ch-Ua")), k) {
			secUaBrowser = k
			break
		}
	}
	//
	deviceType := models.GetDeviceType(
		ctx.Request().Header.Get("User-Agent"),
	)
	//
	// operatorSystem := models.GetOs(ctx.Request().Header.Get("User-Agent"))
	// if operatorSystem == "unknown" &&
	// 	strings.Trim(ctx.Request().Header.Get("Sec-Ch-Ua-Platform"), "\"") != "" {
	// 	operatorSystem = strings.Trim(ctx.Request().Header.Get("Sec-Ch-Ua-Platform"), "\"")
	// }

	deviceInfo := models.GetDeviceInfo(ctx.Request().Header.Get("User-Agent"))
	if err := c.statisticService.CreateAction(
		ctx.Request().Context(),
		models.CreateActionInput{
			SessionId: sessionId,
			MediaId:   mediaId,
			MediaType: payload.MediaType,
			Data:      payload.Data,
		},
		models.SessionInfo{
			SessionId:      sessionId,
			IP:             ctx.RealIP(),
			DeviceType:     deviceType,
			OperatorSystem: deviceInfo.OperatorSystem,
			Browser:        deviceInfo.Browser,
			UserAgent:      ctx.Request().Header.Get("User-Agent"),
			SecUaBrowser:   secUaBrowser,
		},
	); err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, "Create action successfully.")
}

type GetAggregatedMetricsResponse struct {
	Context *models.MetricsContext `json:"context"`
	Data    float64                `json:"data"`
}

type GetAggreatedMetricsMetricsRequest struct {
	From   int64                `json:"from"`
	To     int64                `json:"to"`
	Filter *models.MetricFilter `json:"filter_by"`
}

func (c *StatisticController) GetAggregatedMetrics(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetAggregatedMetrics").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetAggreatedMetricsMetricsRequest
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

	metric := ctx.Param("metric")
	if _, ok := models.ValidAggregatedMetrics[metric]; !ok {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid metric.")
	}

	aggregation := ctx.Param("aggregation")
	if _, ok := models.ValidAggregations[aggregation]; !ok {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid aggregation.")
	}

	if payload.Filter != nil {
		for _, id := range payload.Filter.MediaIds {
			if _, err := uuid.Parse(id); err != nil {
				return response.ResponseFailMessage(
					ctx,
					http.StatusBadRequest,
					"Invalid media id.",
				)
			}
		}

		if payload.Filter.MediaType != "" {
			if payload.Filter.MediaType != models.VideoMediaType &&
				payload.Filter.MediaType != models.AudioMediaType &&
				payload.Filter.MediaType != models.StreamMediaType {
				return response.ResponseFailMessage(
					ctx,
					http.StatusBadRequest,
					"Invalid media type.",
				)
			}
		}

		for _, continent := range payload.Filter.Continents {
			if _, ok := models.ValidContinents[continent]; !ok {
				return response.ResponseFailMessage(
					ctx,
					http.StatusBadRequest,
					"Invalid continent.",
				)
			}
		}

		for _, os := range payload.Filter.OS {
			if _, ok := models.ValidOperatorSystems[os]; !ok {
				return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid os.")
			}
		}

		for _, t := range payload.Filter.DeviceTypes {
			if _, ok := models.ValidDeviceTypes[t]; !ok {
				return response.ResponseFailMessage(
					ctx,
					http.StatusBadRequest,
					"Invalid device type.",
				)
			}
		}

		for _, browser := range payload.Filter.Browsers {
			if _, ok := models.ValidBrowsers[browser]; !ok {
				return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid browser.")
			}
		}

		var tags []string
		for _, tag := range payload.Filter.Tags {
			tags = append(tags, strings.ToLower(tag))
		}

		payload.Filter.Tags = tags
		if !payload.Filter.IsValid(metric, "") {
			return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid filter.")
		}
	}

	allowAggregation := models.MetrixToAggregationMapping[metric]
	if !slices.Contains(allowAggregation, aggregation) {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Invalid aggregation for metric.",
		)
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	metricCtx, result, err := c.statisticService.GetAggregatedMetrics(
		ctx.Request().Context(),
		models.GetAggregatedMetricsInput{
			From:        from,
			To:          to,
			Metric:      metric,
			Aggregation: aggregation,
			Filter:      payload.Filter,
		},
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, GetAggregatedMetricsResponse{
		Context: metricCtx,
		Data:    result,
	})
}

type GetBreakdownMetricsRequest struct {
	From      int64                `json:"from"`
	To        int64                `json:"to"`
	Offset    uint64               `json:"offset"`
	Limit     uint64               `json:"limit"`
	Filter    *models.MetricFilter `json:"filter_by"`
	SortBy    string               `json:"sort_by"`
	OrderBy   string               `json:"order_by"`
	SumOthers bool                 `json:"sum_others"`
}

type GetBreakdownMetricsResponse struct {
	Context *models.MetricsContext `json:"context"`
	Total   int64                  `json:"total"`
	Data    []*models.MetricItem   `json:"data"`
}

func (c *StatisticController) GetBreakdownMetrics(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetBreakdownMetrics").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetBreakdownMetricsRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	var from, to time.Time
	from = time.Unix(payload.From, 0)
	if payload.To != 0 {
		to = time.Unix(payload.To, 0)
	} else {
		to = time.Now().UTC()
	}

	metric := ctx.Param("metric")
	if _, ok := models.ValidBreakdownMetrics[metric]; !ok {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid metric.")
	}

	breakdown := ctx.Param("breakdown")
	if _, ok := models.ValidBreakdowns[breakdown]; !ok {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid breakdown.")
	}

	breakdown = models.BreakdownMapping[breakdown]
	if payload.SortBy != "" {
		if _, ok := models.ValidBreakdownSortBy[payload.SortBy]; !ok {
			return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid sort by.")
		}

		if _, ok := models.OrderMap[payload.OrderBy]; !ok {
			return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid order by.")
		}
	}

	if payload.Filter != nil {
		for _, id := range payload.Filter.MediaIds {
			if _, err := uuid.Parse(id); err != nil {
				return response.ResponseFailMessage(
					ctx,
					http.StatusBadRequest,
					"Invalid media id.",
				)
			}
		}

		for _, continent := range payload.Filter.Continents {
			if _, ok := models.ValidContinents[continent]; !ok {
				return response.ResponseFailMessage(
					ctx,
					http.StatusBadRequest,
					"Invalid continent.",
				)
			}
		}

		for _, os := range payload.Filter.OS {
			if _, ok := models.ValidOperatorSystems[os]; !ok {
				return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid os.")
			}
		}

		for _, browser := range payload.Filter.Browsers {
			if _, ok := models.ValidBrowsers[browser]; !ok {
				return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid browser.")
			}
		}
	}

	if payload.Limit == 0 {
		payload.Limit = 25
	}

	if payload.Limit > models.MaxPageLimit {
		payload.Limit = models.MaxPageLimit
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	metricCtx, data, total, err := c.statisticService.GetBreakdownMetrics(
		ctx.Request().Context(),
		models.GetBreakdownMetricsInput{
			From:      from,
			To:        to,
			Metric:    metric,
			Breakdown: breakdown,
			Filter:    payload.Filter,
			SortBy:    payload.SortBy,
			OrderBy:   payload.OrderBy,
			Offset:    int(payload.Offset),
			Limit:     int(payload.Limit),
			SumOthers: payload.SumOthers,
		},
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, GetBreakdownMetricsResponse{
		Context: metricCtx,
		Total:   total,
		Data:    data,
	},
	)
}

type GetOvertimeMetricsRequest struct {
	From   string               `json:"from"`
	To     string               `json:"to"`
	Limit  uint64               `json:"limit"`
	Offset uint64               `json:"offset"`
	Filter *models.MetricFilter `json:"filter_by"`
}

type GetOvertimeMetricsResponse struct {
	Context *models.MetricsContext `json:"context"`
	Total   int64                  `json:"total"`
	Data    []*models.MetricItem   `json:"data"`
}

// GetStatisticMedias godoc
//
//	@Summary			Get statistic media
//	@Description		Retrieve a list of statistic media
//	@Tags				analytics
//	@Id					GET-analytic-media
//	@Accept				json
//	@Accept				x-www-form-urlencoded
//	@Produce			json
//	@Security			Bearer
//	@Security			BasicAuth
//	@Param				from	query		integer	false	"start time"
//	@Param				to		query		integer	false	"end time"
//	@Param				offset	query		integer	false	"offset, allowed values greater than or equal to 0. Default(0)"	minimum(0)	default(0)
//	@Param				limit	query		integer	false	"results per page. Allowed values 1-100, default is 25"			minimum(1)	maximum(100)	default(25)
//	@Success			200		{object}	GetStatisticMediasResponse
//	@Header				200		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				200		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				200		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			400		{object}	models.ResponseError
//	@Header				400		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				400		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				400		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			403		{object}	models.ResponseError
//	@Header				403		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				403		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				403		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			404		{object}	models.ResponseError
//	@Header				404		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				404		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				404		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Failure			500		{object}	models.ResponseError
//	@Header				500		{integer}	X-RateLimit-Limit		"The request limit per minute"
//	@Header				500		{integer}	X-RateLimit-Remaining	"The number of available requests left for the current time window"
//	@Header				500		{integer}	X-RateLimit-Retry-After	"The number of seconds left until the current rate limit window resets"
//	@Router				/analytics/media [get]
//	@x-group-parameters	true
//	@x-client-paginated	true
//	@x-optional-object	true
//	@x-client-action	"list"
func (c *StatisticController) GetOvertimeMetrics(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetOvertimeMetrics").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetBreakdownMetricsRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	var from, to time.Time
	from = time.Unix(payload.From, 0)
	if payload.To != 0 {
		to = time.Unix(payload.To, 0)
	} else {
		to = time.Now().UTC()
	}

	metric := ctx.Param("metric")
	if _, ok := models.ValidBreakdownMetrics[metric]; !ok {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid metric.")
	}

	interval := ctx.Param("interval")
	if _, ok := models.ValidIntervals[interval]; !ok {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid interval.")
	}

	if payload.Filter != nil {
		for _, id := range payload.Filter.MediaIds {
			if _, err := uuid.Parse(id); err != nil {
				return response.ResponseFailMessage(
					ctx,
					http.StatusBadRequest,
					"Invalid media id.",
				)
			}
		}

		for _, continent := range payload.Filter.Continents {
			if _, ok := models.ValidContinents[continent]; !ok {
				return response.ResponseFailMessage(
					ctx,
					http.StatusBadRequest,
					"Invalid continent.",
				)
			}
		}

		for _, os := range payload.Filter.OS {
			if _, ok := models.ValidOperatorSystems[os]; !ok {
				return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid os.")
			}
		}

		for _, browser := range payload.Filter.Browsers {
			if _, ok := models.ValidBrowsers[browser]; !ok {
				return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid browser.")
			}
		}
	}

	if payload.SortBy != "" {
		if _, ok := models.ValidOvertimeSortBy[payload.SortBy]; !ok {
			return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid sort by.")
		}

		if _, ok := models.OrderMap[payload.OrderBy]; !ok {
			return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid order by.")
		}
	}

	if payload.Limit == 0 {
		payload.Limit = 25
	}

	if payload.Limit > models.MaxPageLimit {
		payload.Limit = models.MaxPageLimit
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	metricCtx, data, total, err := c.statisticService.GetOvertimeMetrics(
		ctx.Request().Context(),
		models.GetOvertimeMetricsInput{
			From:     from,
			To:       to,
			Metric:   metric,
			Interval: interval,
			Filter:   payload.Filter,
			Offset:   int(payload.Offset),
			Limit:    int(payload.Limit),
			SortBy:   payload.SortBy,
			OrderBy:  payload.OrderBy,
		},
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, GetOvertimeMetricsResponse{
		Context: metricCtx,
		Total:   total,
		Data:    data,
	},
	)
}

type GetStatisticMediasRequest struct {
	Limit  int    `json:"limit"  query:"limit"  form:"limit"`
	Offset int    `json:"offset" query:"offset" form:"offset"`
	From   int64  `json:"from"   query:"from"   form:"from"`
	To     int64  `json:"to"     query:"to"     form:"to"`
	Type   string `json:"type"   query:"type"   form:"type"`
}

type GetStatisticMediasResponse struct {
	Total  int64                 `json:"total"`
	Medias []*models.MediaObject `json:"media"`
}

func (c *StatisticController) GetStatisticMedias(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetStatisticMedias").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetStatisticMediasRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	if len(payload.Type) != 0 {
		if payload.Type != models.VideoType &&
			payload.Type != models.AudioType {
			return response.ResponseFailMessage(
				ctx,
				http.StatusBadRequest,
				"Invalid media type.",
			)
		}
	}

	if payload.Limit < 0 {
		payload.Limit = 50
	}

	if payload.Offset < 0 {
		payload.Offset = 0
	}

	if payload.From < 0 || payload.To < 0 {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid time range.")
	}

	if payload.Limit > int(models.MaxPageLimit) {
		payload.Limit = int(models.MaxPageLimit)
	}

	if payload.To == 0 {
		payload.To = time.Now().UTC().Unix()
	}

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	media, total, err := c.statisticService.GetStatisticMedias(
		ctx.Request().Context(),
		models.GetStatisticMediasInput{
			Limit:  payload.Limit,
			Offset: payload.Offset,
			Type:   payload.Type,
			From:   time.Unix(payload.From, 0),
			To:     time.Unix(payload.To, 0),
		},
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, GetStatisticMediasResponse{
		Total:  total,
		Medias: models.NewMediaObjects(media),
	})
}

type GetDataUsageRequest struct {
	From     int64  `json:"from"     query:"from"     form:"from"`
	To       int64  `json:"to"       query:"to"       form:"to"`
	Limit    uint64 `json:"limit"    query:"limit"    form:"limit"`
	Offset   uint64 `json:"offset"   query:"offset"   form:"offset"`
	Interval string `json:"interval" query:"interval" form:"interval"`
}

type GetDataUsageResponse struct {
	Total int64               `json:"total"`
	Data  []*models.DataUsage `json:"data"`
}

func (c *StatisticController) GetDataUsage(
	ctx echo.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetDataUsage").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetDataUsageRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	if payload.From < 0 || payload.To <= 0 {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid time range.")
	}

	if payload.Limit <= 0 {
		payload.Limit = 25
	}

	if payload.Limit > models.MaxPageLimit {
		payload.Limit = models.MaxPageLimit
	}

	if _, ok := models.ValidIntervals[payload.Interval]; !ok {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid interval.")
	}

	from := time.Unix(payload.From, 0)
	to := time.Unix(payload.To, 0)

	authInfo := ctx.Get("authInfo").(models.AuthenticationInfo)
	data, total, err := c.statisticService.GetDataUsage(
		ctx.Request().Context(),
		from,
		to,
		payload.Limit,
		payload.Offset,
		payload.Interval,
		authInfo,
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(ctx, http.StatusOK, GetDataUsageResponse{
		Total: total,
		Data:  data,
	})
}

type GetAdminStatisticRequest struct {
	From int64 `json:"from" query:"from"`
	To   int64 `json:"to"   query:"to"`
}

func (c *StatisticController) GetAdminStatistic(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetAdminStatistic").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetAdminStatisticRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid input.")
	}

	if payload.From < 0 || payload.To <= 0 {
		return response.ResponseFailMessage(ctx, http.StatusBadRequest, "Invalid time range.")
	}

	from := time.Unix(payload.From, 0)
	var to time.Time
	if payload.To != 0 {
		to = time.Unix(payload.To, 0)
	} else {
		to = time.Now().UTC()
	}

	stats, err := c.statisticService.GetAdminStatisticData(
		ctx.Request().Context(),
		models.TimeRange{
			Start: from,
			End:   to,
		},
		ctx.Request().Header.Get("Admin-Api-Key"),
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		stats,
	)
}

type GetMostViewedMediaRequest struct {
	Type  string `json:"type"  form:"type"  query:"type"`
	Limit int    `json:"limit" form:"limit" query:"limit"`
}

func (c *StatisticController) GetMostViewedMedia(ctx echo.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.ApiSum.WithLabelValues("GetMostViewedMedia").
			Observe(time.Since(t).Seconds())
	}()

	var payload GetMostViewedMediaRequest
	if err := ctx.Bind(&payload); err != nil {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Invalid input.",
		)
	}

	if payload.Limit <= 0 || payload.Limit > models.PageSizeLimit {
		payload.Limit = models.DefaultPageLimit
	}

	if payload.Type != models.VideoMediaType && payload.Type != models.StreamMediaType {
		return response.ResponseFailMessage(
			ctx,
			http.StatusBadRequest,
			"Invalid media type.",
		)
	}

	data, err := c.statisticService.GetMostViewedMedia(
		ctx.Request().Context(),
		payload.Type,
		payload.Limit,
		ctx.Request().Header.Get("Admin-Api-Key"),
	)
	if err != nil {
		return response.ResponseError(ctx, err)
	}

	return response.ResponseSuccess(
		ctx,
		http.StatusOK,
		data,
	)
}
