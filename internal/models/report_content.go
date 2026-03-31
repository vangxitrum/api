package models

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ContentReportReason string

const (
	ReasonSexualContent        ContentReportReason = "sexual_content"
	ReasonViolence             ContentReportReason = "violence"
	ReasonHatefulOrAbusive     ContentReportReason = "hateful_or_abusive_content"
	ReasonHarassmentOrBullying ContentReportReason = "harassment_or_bullying"
	ReasonHarmfulOrDangerous   ContentReportReason = "harmful_or_dangerous_acts"
	ReasonMisinformation       ContentReportReason = "misinformation"
	ReasonChildAbuse           ContentReportReason = "child_abuse"
	ReasonPromotesTerrorism    ContentReportReason = "promotes_terrorism"
	ReasonSpamOrMisleading     ContentReportReason = "spam_or_misleading"
	ReasonCopyright            ContentReportReason = "copyright"
	ReasonCaptionsIssue        ContentReportReason = "captions_issue"
)

type ContentReport struct {
	Id          uuid.UUID           `json:"id"          gorm:"type:uuid;primaryKey"`
	ReporterIp  string              `json:"reporter_ip"`
	MediaId     uuid.UUID           `json:"media_id "   gorm:"type:uuid;references:Id"`
	Media       *Media              `json:"-"           gorm:"foreignKey:MediaId"`
	MediaType   string              `json:"media_type"`
	Reason      ContentReportReason `json:"reason"`
	Description string              `json:"description"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Status      string              `json:"status"`
}

type ReportContentRepository interface {
	CreateReport(context.Context, *ContentReport) error
	GetListReports(context.Context, GetContentReportList) ([]*ContentReport, int64, error)
	GetReportByMediaAndIP(context.Context, uuid.UUID, string) (*ContentReport, error)
	GetTotalReportsByMediaId(context.Context, uuid.UUID) (int64, error)
	GetReportById(context.Context, uuid.UUID) (*ContentReport, error)
	UpdateReportStatus(context.Context, uuid.UUID, string) error
	DeleteReport(context.Context, *ContentReport) error
}

func NewContentReport(
	mediaId uuid.UUID,
	reporterIp,
	description,
	mediaType string,
	reason ContentReportReason,
) *ContentReport {
	return &ContentReport{
		Id:          uuid.New(),
		MediaId:     mediaId,
		MediaType:   mediaType,
		ReporterIp:  reporterIp,
		Reason:      reason,
		Description: description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Status:      NewStatus,
	}
}

type GetContentReportList struct {
	SortBy string
	Order  string
	Status string
	Offset uint64
	Limit  uint64
}
