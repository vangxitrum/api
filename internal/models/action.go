package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	PlayActionType            = "play"
	ImpressionPlayActionType  = "impression_play"
	StartActionType           = "start"
	ImpressionStartActionType = "impression_start"
	EndActionType             = "end"
	ImpressionEndActionType   = "impression_end"
	FordwardActionType        = "fwd"
	BackwardActionType        = "bwd"
	StopActionTypes           = "stop"
	ImpressionStopActionType  = "impression_stop"
)

type Action struct {
	Id             uuid.UUID     `json:"id"               gorm:"id;primaryKey;type:uuid"`
	SessionMediaId uuid.UUID     `json:"session_media_id" gorm:"media_id;type:uuid"`
	SessionMedia   *SessionMedia `json:"-"                gorm:"foreignKey:SessionMediaId;references:Id"`
	UserId         uuid.UUID     `json:"user_id"          gorm:"user_id;index:idx_analytic_filter,priority:1;type:uuid"`
	Type           string        `json:"type"             gorm:"index:idx_analytic_filter,priority:2"`
	MediaAt        float64       `json:"media_at"`
	EmittedAt      time.Time     `json:"emitted_at"       gorm:"emitted_at"`
}

func NewAction(sessionMediaId, userId uuid.UUID, actionType string, mediaAt float64) *Action {
	return &Action{
		Id:             uuid.New(),
		SessionMediaId: sessionMediaId,
		UserId:         userId,
		Type:           actionType,
		MediaAt:        mediaAt,
		EmittedAt:      time.Now(),
	}
}

type MetricsContext struct {
	Metric      string        `json:"metric,omitempty"`
	Aggregation string        `json:"aggregation,omitempty"`
	TimeFrame   *TimeFrame    `json:"time_frame,omitempty"`
	Breakdown   string        `json:"breakdown,omitempty"`
	Interval    string        `json:"interval,omitempty"`
	Filter      *MetricFilter `json:"filter,omitempty"`
}

type TimeFrame struct {
	From time.Time `json:"from,omitempty"`
	To   time.Time `json:"to,omitempty"`
}

type CreateActionInput struct {
	Data      []*ActionItem
	MediaType string
	MediaId   uuid.UUID
	SessionId uuid.UUID
}

type ActionItem struct {
	Type    string  `json:"type"`
	MediaAt float64 `json:"media_at"`
}

type GetAggregatedMetricsInput struct {
	From, To                    time.Time
	Metric, Aggregation, SortBy string
	Offset, Limit               int
	Filter                      *MetricFilter
}

type MetricItem struct {
	MetricValue    float64   `json:"metric_value"`
	DimensionValue string    `json:"dimension_value,omitempty"`
	EmittedAt      time.Time `json:"emitted_at,omitempty"`
}

type MetricFilter struct {
	MediaIds    []string `json:"media_ids,omitempty"`
	MediaType   string   `json:"media_type,omitempty"`
	Continents  []string `json:"continents,omitempty"`
	Countries   []string `json:"countries,omitempty"`
	DeviceTypes []string `json:"device_types,omitempty"`
	OS          []string `json:"os,omitempty"`
	Browsers    []string `json:"browsers,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func (m *MetricFilter) IsValid(metric, breakdown string) bool {
	if metric == PlayMetric ||
		(metric == PlayTotalMetric && breakdown == MediaIdBreakdown) {
		if len(m.Continents) > 0 ||
			len(m.Countries) > 0 ||
			len(m.DeviceTypes) > 0 ||
			len(m.OS) > 0 ||
			len(m.Browsers) > 0 ||
			len(m.Tags) > 0 {
			return false
		}
	}

	return true
}

func (m *MetricFilter) BuildQuery() string {
	convertToQuery := func(values []string) string {
		if len(values) == 0 {
			return ""
		}

		return fmt.Sprintf("'%s'", strings.Join(values, "','"))
	}

	if m == nil {
		return ""
	}

	query := ""
	if len(m.MediaIds) > 0 {
		query += fmt.Sprintf("where media_id in (%s)", convertToQuery(m.MediaIds))
	}

	if len(m.Continents) > 0 {
		if query != "" {
			query += " and "
		} else {
			query += "where "
		}

		query += fmt.Sprintf("s.continent in (%s)", convertToQuery(m.Continents))
	}

	if len(m.OS) > 0 {
		if query != "" {
			query += " and "
		} else {
			query += "where "
		}

		query += fmt.Sprintf("s.os in (%s)", convertToQuery(m.OS))
	}

	if len(m.Browsers) > 0 {
		if query != "" {
			query += " and "
		} else {
			query += "where "
		}

		query += fmt.Sprintf("s.browser in (%s)", convertToQuery(m.Browsers))
	}

	if len(m.Countries) > 0 {
		if query != "" {
			query += " and "
		} else {
			query += "where "
		}

		query += fmt.Sprintf("s.country in (%s)", convertToQuery(m.Countries))
	}

	if len(m.DeviceTypes) > 0 {
		if query != "" {
			query += " and "
		} else {
			query += "where "
		}

		query += fmt.Sprintf("s.device_type in (%s)", convertToQuery(m.DeviceTypes))
	}

	if len(m.Tags) > 0 {
		if query != "" {
			query += " and "
		} else {
			query += "where "
		}

		query += fmt.Sprintf(
			"v.tags ilike '%%%s%%'",
			strings.Join(m.Tags, ","),
		)
	}

	return query
}

type GetBreakdownMetricsInput struct {
	From, To          time.Time
	Metric, Breakdown string
	Offset, Limit     int
	SortBy            string
	OrderBy           string
	Filter            *MetricFilter
	SumOthers         bool
}

func (input *GetBreakdownMetricsInput) Hash(id uuid.UUID) (string, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	data = append(data, "breakdown"...)
	return base64.StdEncoding.EncodeToString(data), nil
}

type MetricCacheItem struct {
	Rs    []*MetricItem
	Total int64
}

type GetOvertimeMetricsInput struct {
	From, To      time.Time
	Metric        string
	Interval      string
	Filter        *MetricFilter
	SortBy        string
	OrderBy       string
	Offset, Limit int
}

func (input *GetOvertimeMetricsInput) Hash(id uuid.UUID) (string, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	data = append(data, "overtime"...)
	return base64.StdEncoding.EncodeToString(data), nil
}
