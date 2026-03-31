package models

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

var (
	WebhookSortBy = []string{
		"created_at",
		"url",
		"name",
	}
	EventEncodingFinished = "encoding.finished"
	EventEncodingStarted  = "encoding.started"
	EventFileReceived     = "file.received"
	EventPartialFinished  = "partial.finished"
	EventEncodingFailed   = "encoding.failed"
)

type WebhookRepository interface {
	CreateWebhook(
		context.Context, *Webhook,
	) (*Webhook, error)

	CreateWebhookRetry(
		context.Context, *WebhookRetry,
	) error

	UpdateWebhookRetry(
		context.Context, *WebhookRetry,
	) error

	DeleteWebhookRetry(
		context.Context, uuid.UUID,
	) error

	GetWebhookRetryList(
		context.Context,
	) ([]*WebhookRetry, error)

	GetUserWebhookList(
		context.Context, GetWebhookListFilter,
	) ([]*Webhook, int64, error)
	GetUserWebhookById(
		context.Context, uuid.UUID, uuid.UUID,
	) (*Webhook, error)

	GetWebhookById(
		context.Context, uuid.UUID,
	) (*Webhook, error)

	GetWebhookListBetweenTime(
		context.Context, uuid.UUID, time.Time,
		time.Time,
	) ([]*Webhook, error)
	GetFileReceivedWebhooksByUserId(
		context.Context,
		uuid.UUID,
	) ([]*Webhook, error)
	GetEncodingStartedWebhooksByUserId(
		context.Context, uuid.UUID,
	) ([]*Webhook, error)
	GetEncodingFinishedWebhooksByUserId(
		context.Context, uuid.UUID,
	) ([]*Webhook, error)
	GetEncodingFailWebhooksByUserId(
		context.Context, uuid.UUID,
	) ([]*Webhook, error)
	GetPartialWebhookByUserId(
		context.Context, uuid.UUID,
	) ([]*Webhook, error)

	UpdateUserWebhook(
		context.Context, UpdateWebhookInput,
	) error
	UpdateWebhookTriggeredAt(context.Context, uuid.UUID, uuid.UUID) error

	DeleteWebhookById(
		context.Context, uuid.UUID, uuid.UUID,
	) error
	DeleteUserWebhooks(context.Context, uuid.UUID) error
}

type Webhook struct {
	Id                    uuid.UUID `json:"id"                gorm:"primaryKey;id;type:uuid"`
	UserId                uuid.UUID `json:"user_id"           gorm:"type:uuid"`
	Name                  string    `json:"name"`
	Url                   string    `json:"url"`
	EventFileReceived     bool      `json:"file_received"`
	EventEncodingStarted  bool      `json:"encoding_started"`
	EventEncodingFinished bool      `json:"encoding_finished"`
	EventEncodingFailed   bool      `json:"encoding_failed"`
	EventPartialFinished  bool      `json:"partial_finished"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	LastTriggeredAt       time.Time `json:"last_triggered_at"`
} //	@name	Webhook

type WebhookRetry struct {
	WebhookId    uuid.UUID       `json:"webhook_id"    gorm:"type:uuid"`
	Webhook      *Webhook        `json:"webhook"       gorm:"foreignKey:WebhookId;constraint:OnDelete:CASCADE"`
	Notification json.RawMessage `json:"notification"`
	RetryCount   int             `json:"retry_count"`
	NextRetryAt  time.Time       `json:"next_retry_at"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type CreateWebhookInput struct {
	Name                  string
	UserId                uuid.UUID
	Url                   string
	EventFileReceived     bool
	EventEncodingStarted  bool
	EventEncodingFinished bool
	EventEncodingFailed   bool
	EventPartialFinished  bool
}

type DeleteWebhookInput struct {
	UserId uuid.UUID
	Id     uuid.UUID
}

func NewWebhook(
	userId uuid.UUID,
	name, url string,
	encodingStarted, encodingFinished, fileReceived, encodingFailed, partialFinished bool,
) *Webhook {
	return &Webhook{
		Id:                    uuid.New(),
		UserId:                userId,
		Url:                   url,
		Name:                  name,
		EventFileReceived:     fileReceived,
		EventEncodingStarted:  encodingStarted,
		EventEncodingFinished: encodingFinished,
		EventEncodingFailed:   encodingFailed,
		EventPartialFinished:  partialFinished,
		CreatedAt:             time.Now().UTC(),
		UpdatedAt:             time.Now().UTC(),
		LastTriggeredAt:       time.Now().UTC(),
	}
}

func NewWebhookRetry(
	webhookID uuid.UUID,
	Notification json.RawMessage,
) *WebhookRetry {
	return &WebhookRetry{
		WebhookId:    webhookID,
		Notification: Notification,
		RetryCount:   0,
		NextRetryAt:  time.Now().UTC().Add(time.Second * 30),
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
}

type GetWebhookListFilter struct {
	UserId                uuid.UUID
	EventFileReceived     bool   `json:"file_received"     query:"file_received"`
	EventEncodingStarted  bool   `json:"encoding_started"  query:"encoding_started"`
	EventEncodingFinished bool   `json:"encoding_finished" query:"encoding_finished"`
	EventEncodingFailed   bool   `json:"encoding_failed"   query:"encoding_failed"`
	EventPartialFinished  bool   `json:"partial_finished"  query:"partial_finished"`
	Offset                uint64 `json:"offset"            query:"offset"`
	Limit                 uint64 `json:"limit"             query:"limit"`
	SortBy                string `json:"sort_by"           query:"sort_by"`
	Search                string `json:"search"            query:"search"`
	Order                 string `json:"order_by"          query:"order_by"`
}
type UpdateWebhookInput struct {
	Id                    uuid.UUID
	UserId                uuid.UUID
	Name                  *string
	Url                   *string
	EventFileReceived     *bool
	EventEncodingStarted  *bool
	EventEncodingFinished *bool
	EventEncodingFailed   *bool
	EventPartialFinished  *bool
}

type WebhookNotification struct {
	Type      string    `json:"type"`
	EmittedAt time.Time `json:"emitted_at"`
	MediaId   uuid.UUID `json:"media_id"`
	Qualities string    `json:"qualities"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	MediaType string    `json:"media_type"`
	UserId    uuid.UUID `json:"user_id"`
}
