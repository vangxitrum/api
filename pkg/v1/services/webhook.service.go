package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/rabbitmq"
)

type WebhookService struct {
	webhookRepo       models.WebhookRepository
	resty             *resty.Client
	callWebhookWorker *rabbitmq.RabbitMQ
}

func NewWebhookService(
	webhookRepo models.WebhookRepository,
	resty *resty.Client,
	callWebhookWorker *rabbitmq.RabbitMQ,
) *WebhookService {
	webhookService := &WebhookService{
		webhookRepo:       webhookRepo,
		resty:             resty,
		callWebhookWorker: callWebhookWorker,
	}
	webhookService.startCallWebhookWorker(context.Background())
	return webhookService
}

func (s *WebhookService) CreateWebhook(
	ctx context.Context,
	input models.CreateWebhookInput,
) (*models.Webhook, error) {
	result, err := s.webhookRepo.CreateWebhook(
		ctx, models.NewWebhook(
			input.UserId,
			input.Name,
			input.Url,
			input.EventEncodingStarted,
			input.EventEncodingFinished,
			input.EventFileReceived,
			input.EventEncodingFailed,
			input.EventPartialFinished,
		),
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return result, nil
}

func (s *WebhookService) GetWebhookList(
	ctx context.Context,
	input models.GetWebhookListFilter,
) ([]*models.Webhook, int64, error) {
	var result []*models.Webhook
	webhooks, total, err := s.webhookRepo.GetUserWebhookList(
		ctx, models.GetWebhookListFilter{
			UserId:                input.UserId,
			EventEncodingStarted:  input.EventEncodingStarted,
			EventEncodingFinished: input.EventEncodingFinished,
			EventFileReceived:     input.EventFileReceived,
			EventEncodingFailed:   input.EventEncodingFailed,
			EventPartialFinished:  input.EventPartialFinished,
			Offset:                input.Offset,
			Order:                 input.Order,
			Limit:                 input.Limit,
			SortBy:                input.SortBy,
			Search:                input.Search,
		},
	)
	if err != nil {
		return nil, 0, response.NewInternalServerError(err)
	}
	for _, webhook := range webhooks {
		result = append(result, webhook)
	}
	return webhooks, total, nil
}

func (s *WebhookService) DeleteUserWebhook(
	ctx context.Context, userId, id uuid.UUID,
) error {
	if err := s.webhookRepo.DeleteWebhookById(
		ctx,
		userId,
		id,
	); err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *WebhookService) GetWebhookById(
	ctx context.Context, userId, id uuid.UUID,
) (*models.Webhook, error) {
	result, err := s.webhookRepo.GetUserWebhookById(
		ctx,
		userId,
		id,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	return result, nil
}

func (s *WebhookService) UpdateWebhook(
	ctx context.Context,
	input models.UpdateWebhookInput,
) error {
	if err := s.webhookRepo.UpdateUserWebhook(
		ctx, models.UpdateWebhookInput{
			Id:                    input.Id,
			UserId:                input.UserId,
			Url:                   input.Url,
			Name:                  input.Name,
			EventFileReceived:     input.EventFileReceived,
			EventEncodingStarted:  input.EventEncodingStarted,
			EventEncodingFinished: input.EventEncodingFinished,
		},
	); err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *WebhookService) handleWebhook(
	ctx context.Context,
	webhook *models.Webhook,
	notify models.WebhookNotification,
) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer cancel()

	resp, err := s.resty.R().
		SetContext(ctx).
		SetContentLength(true).
		SetBody(notify).
		Post(webhook.Url)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	if resp.StatusCode() >= http.StatusOK && resp.StatusCode() < http.StatusMultipleChoices {
		if err := s.webhookRepo.UpdateWebhookTriggeredAt(
			ctx,
			webhook.UserId,
			webhook.Id,
		); err != nil {
			return response.NewInternalServerError(err)
		}
		return nil
	}

	return nil
}

func (s *WebhookService) callWebhook(
	ctx context.Context,
	notify models.WebhookNotification,
) error {
	qualities := strings.Split(notify.Qualities, ",")
	sort.Slice(qualities, func(i, j int) bool {
		qualityI, _ := strconv.Atoi(strings.TrimSuffix(qualities[i], "p"))
		qualityJ, _ := strconv.Atoi(strings.TrimSuffix(qualities[j], "p"))
		return qualityI < qualityJ
	})

	notify.Qualities = strings.Join(qualities, ",")

	var (
		webhooks []*models.Webhook
		err      error
	)
	switch notify.Type {
	case models.EventFileReceived:
		webhooks, err = s.webhookRepo.GetFileReceivedWebhooksByUserId(
			ctx,
			notify.UserId,
		)
		if err != nil {
			return err
		}
	case models.EventEncodingStarted:
		webhooks, err = s.webhookRepo.GetEncodingStartedWebhooksByUserId(
			ctx,
			notify.UserId,
		)
		if err != nil {
			return err
		}
	case models.EventEncodingFinished:
		webhooks, err = s.webhookRepo.GetEncodingFinishedWebhooksByUserId(
			ctx,
			notify.UserId,
		)
		if err != nil {
			return err
		}
	case models.EventEncodingFailed:
		webhooks, err = s.webhookRepo.GetEncodingFailWebhooksByUserId(
			ctx,
			notify.UserId,
		)
		if err != nil {
			return err
		}
	case models.EventPartialFinished:
		webhooks, err = s.webhookRepo.GetPartialWebhookByUserId(
			ctx,
			notify.UserId,
		)
		if err != nil {
			return err
		}
	}

	for _, webhook := range webhooks {
		if err := s.handleWebhook(ctx, webhook, notify); err != nil {
			slog.Error("Failed to handle webhook",
				"error", err,
				"webhook_url", webhook.Url,
				"user_id", webhook.UserId,
				"webhook_id", webhook.Id)

			notifyJSON, err := json.Marshal(notify)
			if err != nil {
				slog.Error("Failed to marshal webhook notification")
				continue
			}

			retryWebhook := models.NewWebhookRetry(
				webhook.Id,
				notifyJSON,
			)
			if err := s.webhookRepo.CreateWebhookRetry(
				ctx,
				retryWebhook,
			); err != nil {
				slog.Error("Failed to create webhook retry")
			}

			continue
		}
	}
	return nil
}

func (s *WebhookService) startCallWebhookWorker(ctx context.Context) {
	messCh, err := s.callWebhookWorker.Consume("server-call-webhook")
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			for mess := range messCh {
				var notify models.WebhookNotification
				if err := json.Unmarshal(
					mess.Body,
					&notify,
				); err != nil {
					slog.Error(
						"callWebhook, unmarshal message error",
						slog.Any("error", err),
					)
					continue
				}

				if err := s.callWebhook(
					ctx,
					notify,
				); err != nil {
					slog.Error(
						"callWebhook, call webhook error",
						slog.Any("error", err),
					)
					continue
				}
			}
		}
	}()
}

func (s *WebhookService) CheckWebhookById(
	ctx context.Context,
	userId, webhookId uuid.UUID,
) error {
	webhook, err := s.webhookRepo.GetUserWebhookById(
		ctx,
		userId,
		webhookId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		1000*time.Millisecond,
	)
	defer cancel()
	notify := models.WebhookNotification{
		Type:      models.EventEncodingStarted,
		EmittedAt: time.Now().UTC(),
		MediaId:   uuid.UUID{},
		Qualities: "720p",
		Title:     "Test.mp4",
		UserId:    userId,
		MediaType: models.VideoMediaType,
	}
	res, err := s.resty.R().
		SetContext(ctx).
		SetContentLength(true).
		SetBody(notify).
		Post(webhook.Url)
	if err != nil {
		return response.NewBadRequestError("Invalid url.")
	}

	if res.StatusCode() != http.StatusOK {
		return response.NewHttpError(
			res.StatusCode(),
			fmt.Errorf(
				"Webhook response code %d.",
				res.StatusCode()),
		)
	}

	if err := s.webhookRepo.UpdateWebhookTriggeredAt(
		ctx, webhook.UserId, webhook.Id); err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}
	return nil
}

func (s *WebhookService) HandleWebhookRetry(ctx context.Context) error {
	retries, err := s.webhookRepo.GetWebhookRetryList(ctx)

	retryDelaysMap := map[int]time.Duration{
		1: 1 * time.Minute,
		2: 2 * time.Minute,
		3: 4 * time.Minute,
		4: 8 * time.Minute,
		5: 16 * time.Minute,
	}

	if err != nil {
		return err
	}

	if len(retries) == 0 {
		return nil
	}

	for _, retry := range retries {
		if time.Now().Before(retry.NextRetryAt) {
			continue
		}

		webhook, err := s.webhookRepo.GetWebhookById(ctx, retry.WebhookId)
		if err != nil {
			if errors.Is(
				err,
				gorm.ErrRecordNotFound,
			) {
				if err := s.webhookRepo.DeleteWebhookRetry(ctx, retry.WebhookId); err != nil {
					slog.Error("Failed to delete webhook retry", "error", err)
				}
				continue
			}
			slog.Error("Failed to get webhook", "error", err)
			continue
		}

		resp, err := s.resty.R().
			SetContext(ctx).
			SetContentLength(true).
			SetBody(retry.Notification).
			Post(webhook.Url)
		if err != nil {
			retry.RetryCount++
			if nextDelay, exists := retryDelaysMap[retry.RetryCount]; exists {
				retry.NextRetryAt = time.Now().Add(nextDelay)
				if err := s.webhookRepo.UpdateWebhookRetry(ctx, retry); err != nil {
					slog.Error("Failed to update webhook retry", "error", err)
				}
			} else {
				if err := s.webhookRepo.DeleteWebhookRetry(ctx, retry.WebhookId); err != nil {
					slog.Error("Failed to delete webhook retry", "error", err)
				}
			}
			continue
		}

		if resp.StatusCode() >= http.StatusOK && resp.StatusCode() < http.StatusMultipleChoices {
			if err := s.webhookRepo.DeleteWebhookRetry(ctx, retry.WebhookId); err != nil {
				slog.Error("Failed to delete webhook retry", "error", err)
			}
			if err := s.webhookRepo.UpdateWebhookTriggeredAt(ctx, webhook.UserId, webhook.Id); err != nil {
				slog.Error("Failed to update webhook triggered time", "error", err)
			}
		} else {
			retry.RetryCount++
			if nextDelay, exists := retryDelaysMap[retry.RetryCount]; exists {
				retry.NextRetryAt = time.Now().Add(nextDelay)
				if err := s.webhookRepo.UpdateWebhookRetry(ctx, retry); err != nil {
					slog.Error("Failed to update webhook retry", "error", err)
				}
			} else {
				if err := s.webhookRepo.DeleteWebhookRetry(ctx, retry.WebhookId); err != nil {
					slog.Error("Failed to delete webhook retry", "error", err)
				}
			}
		}
	}
	return nil
}

func (s *WebhookService) DeleteUserWebhooks(
	ctx context.Context,
	userId uuid.UUID,
) error {
	if err := s.webhookRepo.DeleteUserWebhooks(ctx, userId); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}
