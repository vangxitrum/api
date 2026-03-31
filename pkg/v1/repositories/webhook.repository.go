package repositories

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type webhookRepository struct {
	db *gorm.DB
}

func MustNewWebhookRepository(
	db *gorm.DB, init bool,
) models.WebhookRepository {
	if init {
		err := db.AutoMigrate(&models.Webhook{}, &models.WebhookRetry{})
		if err != nil {
			panic(err)
		}
	}
	return &webhookRepository{
		db: db,
	}
}

func (r *webhookRepository) CreateWebhook(
	ctx context.Context, webhook *models.Webhook,
) (*models.Webhook, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateWebhook").Observe(time.Since(t).Seconds())
	}()
	if err := r.db.WithContext(ctx).Create(webhook).Error; err != nil {
		return nil, err
	}

	return webhook, nil
}

func (r *webhookRepository) CreateWebhookRetry(
	ctx context.Context, retry *models.WebhookRetry,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateWebhookRetry").
			Observe(time.Since(t).Seconds())
	}()
	if err := r.db.WithContext(ctx).Create(retry).Error; err != nil {
		return err
	}

	return nil
}

func (r *webhookRepository) GetUserWebhookList(
	ctx context.Context,
	filter models.GetWebhookListFilter,
) ([]*models.Webhook, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserWebhookList").
			Observe(time.Since(t).Seconds())
	}()

	var (
		result []*models.Webhook
		total  int64
	)
	condition := fmt.Sprintf(
		"user_id = '%s'",
		filter.UserId,
	)
	if filter.EventFileReceived {
		condition += " and event_file_received = true"
	}

	if filter.EventEncodingStarted {
		condition += " and event_encoding_started = true"
	}

	if filter.EventEncodingFinished {
		condition += " and event_encoding_finished = true"
	}

	if filter.EventEncodingFailed {
		condition += " and event_encoding_failed = true"
	}

	if filter.EventPartialFinished {
		condition += " and event_partial_finished = true"
	}

	query := r.db.WithContext(ctx).Model(&models.Webhook{}).Where(condition)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Search != "" {
		query = query.Where(
			"name ILIKE ?",
			"%"+filter.Search+"%",
		)
	}
	if err := query.Offset(
		int(filter.Offset),
	).
		Limit(int(filter.Limit)).
		Order(
			fmt.Sprintf(
				"%s %s",
				filter.SortBy,
				filter.Order,
			),
		).
		Find(&result).Error; err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (r *webhookRepository) GetUserWebhookById(
	ctx context.Context, userId uuid.UUID,
	webhookId uuid.UUID,
) (*models.Webhook, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserWebhookById").
			Observe(time.Since(t).Seconds())
	}()

	var result models.Webhook

	if err := r.db.WithContext(ctx).Where(
		"id = ? and user_id = ?",
		webhookId,
		userId,
	).
		First(&result).Error; err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *webhookRepository) GetWebhookById(
	ctx context.Context, webhookId uuid.UUID,
) (*models.Webhook, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetWebhookById").
			Observe(time.Since(t).Seconds())
	}()

	var result models.Webhook

	if err := r.db.WithContext(ctx).Where(
		"id = ?",
		webhookId,
	).First(&result).Error; err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *webhookRepository) GetWebhookListBetweenTime(
	ctx context.Context, userId uuid.UUID,
	from time.Time, to time.Time,
) ([]*models.Webhook, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetWebhookListBetweenTime").
			Observe(time.Since(t).Seconds())
	}()

	var result []*models.Webhook

	if err := r.db.WithContext(ctx).Where(
		"user_id = ? and created_at >= ? and created_at <= ?",
		userId,
		from,
		to,
	).
		Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *webhookRepository) GetWebhookRetryList(
	ctx context.Context,
) ([]*models.WebhookRetry, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetWebhookRetryList").
			Observe(time.Since(t).Seconds())
	}()

	var result []*models.WebhookRetry

	if err := r.db.WithContext(ctx).
		Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *webhookRepository) GetEncodingStartedWebhooksByUserId(
	ctx context.Context, userId uuid.UUID,
) ([]*models.Webhook, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetEncodingStartedWebhooksByUserId").
			Observe(time.Since(t).Seconds())
	}()

	var result []*models.Webhook

	if err := r.db.WithContext(ctx).Where(
		"user_id = ? and event_encoding_started = true",
		userId,
	).
		Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *webhookRepository) GetEncodingFailWebhooksByUserId(
	ctx context.Context, userId uuid.UUID,
) ([]*models.Webhook, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetEncodingFailWebhooksByUserId").
			Observe(time.Since(t).Seconds())
	}()

	var result []*models.Webhook

	if err := r.db.WithContext(ctx).Where(
		"user_id = ? and event_encoding_failed = true",
		userId,
	).
		Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *webhookRepository) GetPartialWebhookByUserId(
	ctx context.Context, userId uuid.UUID,
) ([]*models.Webhook, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetPartialWebhookByUserId").
			Observe(time.Since(t).Seconds())
	}()

	var result []*models.Webhook

	if err := r.db.WithContext(ctx).Where(
		"user_id = ? and event_partial_finished = true",
		userId,
	).
		Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *webhookRepository) GetEncodingFinishedWebhooksByUserId(
	ctx context.Context, userId uuid.UUID,
) ([]*models.Webhook, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetEncodingFinishedWebhooksByUserId").
			Observe(time.Since(t).Seconds())
	}()

	var result []*models.Webhook

	if err := r.db.WithContext(ctx).Where(
		"user_id = ? and event_encoding_finished = true",
		userId,
	).
		Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *webhookRepository) GetFileReceivedWebhooksByUserId(
	ctx context.Context, userId uuid.UUID,
) ([]*models.Webhook, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetFileReceivedWebhooksByUserId").
			Observe(time.Since(t).Seconds())
	}()

	var result []*models.Webhook

	if err := r.db.WithContext(ctx).Where(
		"user_id = ? and event_file_received = true",
		userId,
	).
		Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *webhookRepository) DeleteWebhookById(
	ctx context.Context, userId uuid.UUID,
	webhookId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteWebhookById").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).
		Delete(&models.Webhook{}, "id = ? AND user_id = ?", webhookId, userId)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *webhookRepository) DeleteUserWebhooks(ctx context.Context, userId uuid.UUID) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteUserWebhooks").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).
		Joins("JOIN webhooks ON webhooks.id = webhook_retries.webhook_id").
		Where("webhooks.user_id = ?", userId).
		Delete(&models.WebhookRetry{}).Error; err != nil {
		slog.Error("Failed to delete webhook retry", "error", err)
	}

	if err := r.db.WithContext(ctx).Where("user_id = ?", userId).Delete(&models.Webhook{}).Error; err != nil {
		return err
	}

	return nil
}

func (r *webhookRepository) UpdateUserWebhook(
	ctx context.Context,
	input models.UpdateWebhookInput,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateUserWebhook").
			Observe(time.Since(t).Seconds())
	}()

	updateMap := map[string]any{}
	if input.Name != nil {
		updateMap["name"] = *input.Name
	}
	if input.EventEncodingFinished != nil {
		updateMap["event_encoding_finished"] = *input.EventEncodingFinished
	}
	if input.EventEncodingStarted != nil {
		updateMap["event_encoding_started"] = *input.EventEncodingStarted
	}
	if input.EventFileReceived != nil {
		updateMap["event_file_received"] = *input.EventFileReceived
	}
	if input.EventEncodingFailed != nil {
		updateMap["event_encoding_failed"] = *input.EventEncodingFailed
	}
	if input.EventPartialFinished != nil {
		updateMap["event_partial_finished"] = *input.EventPartialFinished
	}
	if input.Url != nil {
		updateMap["url"] = *input.Url
	}

	result := r.db.WithContext(ctx).Model(&models.Webhook{}).Where(
		"id = ? and user_id = ?",
		input.Id,
		input.UserId,
	).Updates(updateMap)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *webhookRepository) UpdateWebhookRetry(
	ctx context.Context,
	retry *models.WebhookRetry,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateWebhookRetry").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).Model(&models.WebhookRetry{}).Where(
		"webhook_id = ? ",
		retry.WebhookId,
	).Updates(retry)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *webhookRepository) DeleteWebhookRetry(
	ctx context.Context,
	webhookId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteWebhookRetry").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).Delete(&models.WebhookRetry{}, "webhook_id = ?", webhookId)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *webhookRepository) UpdateWebhookTriggeredAt(
	ctx context.Context,
	userId uuid.UUID,
	webhookId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateWebhookTriggeredAt").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).Model(&models.Webhook{}).Where(
		"id = ? and user_id = ?",
		webhookId,
		userId,
	).Updates(
		map[string]interface{}{
			"last_triggered_at": time.Now().UTC(),
		},
	)

	if result.Error != nil {
		return result.Error
	}

	return nil
}
