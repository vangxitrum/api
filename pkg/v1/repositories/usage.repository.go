package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type UsageRepository struct {
	db *gorm.DB
}

func MustNewUsageRepository(db *gorm.DB, init bool) models.UsageRepository {
	if init {
		if err := db.AutoMigrate(&models.Usage{}, &models.UsageLog{}); err != nil {
			panic("failed to migrate usage models")
		}
	}

	return &UsageRepository{
		db: db,
	}
}

func (r *UsageRepository) Create(
	ctx context.Context,
	usage *models.Usage,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateUsage").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(usage).Error
}

func (r *UsageRepository) CreateLog(
	ctx context.Context,
	log *models.UsageLog,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateUsageLog").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(log).Error
}

func (r *UsageRepository) GetUsageLogs(
	ctx context.Context,
	cursor time.Time,
) ([]*models.UsageLog, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUsageLogs").
			Observe(time.Since(now).Seconds())
	}()

	var logs []*models.UsageLog
	if err := r.db.WithContext(ctx).
		Raw(`
			SELECT user_id
				,coalesce(sum(storage), 0) as storage
				,coalesce(sum(transcode), 0) as transcode
				,coalesce(sum(delivery), 0) as delivery
				,coalesce(sum(cast(transcode_cost as numeric)),0) as transcode_cost
				,created_at
			FROM usage_logs
			WHERE created_at < ? and is_user_cost = true
			GROUP BY user_id, created_at
		`, cursor).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *UsageRepository) GetUserLatestUsage(
	ctx context.Context,
	userId uuid.UUID,
	cursor time.Time,
) (*models.Usage, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserLatestUsage").
			Observe(time.Since(now).Seconds())
	}()

	var usage models.Usage
	if err := r.db.WithContext(ctx).
		Where("user_id = ? and created_at = ?", userId, cursor).
		First(&usage).Error; err != nil {
		return nil, err
	}

	return &usage, nil
}

func (r *UsageRepository) GetLastHourUsagesByStatus(
	ctx context.Context,
	status string,
) ([]*models.Usage, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLastHourUsagesByStatus").
			Observe(time.Since(now).Seconds())
	}()

	var usages []*models.Usage
	if err := r.db.WithContext(ctx).
		Where(
			"status = ? and created_at <= ?",
			status,
			time.Now().UTC().Add(-1*time.Hour).Truncate(time.Hour),
		).
		Order("created_at").
		Find(&usages).Error; err != nil {
		return nil, err
	}

	return usages, nil
}

func (r *UsageRepository) GetUserUsagesByStatus(
	ctx context.Context,
	userId uuid.UUID,
	status string,
) ([]*models.Usage, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserUsagesByStatus").
			Observe(time.Since(now).Seconds())
	}()

	var usages []*models.Usage
	if err := r.db.WithContext(ctx).
		Where(
			"status = ? and user_id = ?",
			status,
			userId,
		).
		Order("created_at").
		Find(&usages).Error; err != nil {
		return nil, err
	}

	return usages, nil
}

func (r *UsageRepository) GetLastHourUsages(
	ctx context.Context,
) ([]*models.Usage, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLastHourUsages").
			Observe(time.Since(now).Seconds())
	}()

	var currentUsage []*models.Usage
	timeCursor := now.Truncate(time.Hour).Add(-(1 * time.Hour))
	if err := r.db.Model(&models.Usage{}).Where("created_at = ?", timeCursor).Find(&currentUsage).Error; err != nil {
		return nil, err
	}

	return currentUsage, nil
}

func (r *UsageRepository) UpdateUsage(
	ctx context.Context,
	usage *models.Usage,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateUsage").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Model(usage).
		Updates(usage).Error
}

func (r *UsageRepository) DeleteUsageLogsByCursor(
	ctx context.Context,
	cursor time.Time,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteUsageLogsByCursor").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Delete(&models.UsageLog{}, "created_at < ?", cursor).Error
}

func (r *UsageRepository) GetUsageLogsByCursor(
	ctx context.Context,
	cursor time.Time,
) ([]*models.UsageLog, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUsageLogsByCursor").
			Observe(time.Since(now).Seconds())
	}()

	var logs []*models.UsageLog
	if err := r.db.WithContext(ctx).
		Raw(`
            SELECT created_at
                ,coalesce(sum(storage), 0) as storage
                ,coalesce(sum(transcode), 0) as transcode
                ,coalesce(sum(delivery), 0) as delivery
                ,coalesce(sum(cast(transcode_cost as numeric)),0) as transcode_cost
            FROM usage_logs
            WHERE created_at < ?
            GROUP BY created_at
        `, cursor).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *UsageRepository) GetUserUsageByTimeRange(
	ctx context.Context,
	userId uuid.UUID,
	timeRange models.TimeRange,
) (decimal.Decimal, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserUsageByTimeRange").
			Observe(time.Since(now).Seconds())
	}()

	var totalCost decimal.Decimal

	err := r.db.WithContext(ctx).
		Model(&models.Usage{}).
		Select("COALESCE(SUM(CAST(storage_cost AS numeric) + CAST(delivery_cost AS numeric)), 0) as total_cost").
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userId, timeRange.Start, timeRange.End).
		Scan(&totalCost).Error
	if err != nil {
		return decimal.Zero, err
	}
	return totalCost, nil
}

func (r *UsageRepository) GetUserLastHourUsage(
	ctx context.Context,
	userId uuid.UUID,
) (decimal.Decimal, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserLastHourUsage").
			Observe(time.Since(now).Seconds())
	}()

	var totalCost decimal.Decimal
	timeCursor := now.Truncate(time.Hour).Add(-(1 * time.Hour))

	err := r.db.WithContext(ctx).
		Model(&models.Usage{}).
		Select("CAST(total_cost AS numeric) as total_cost").
		Where("user_id = ? AND created_at = ?", userId, timeCursor).
		Scan(&totalCost).Error

	return totalCost, err
}

func (r *UsageRepository) GetUserIdsByTimeRange(
	ctx context.Context,
	timeRange models.TimeRange,
) ([]uuid.UUID, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserIdsByTimeRange").
			Observe(time.Since(now).Seconds())
	}()

	var userIds []uuid.UUID

	err := r.db.WithContext(ctx).
		Model(&models.Usage{}).
		Distinct("user_id").
		Where("created_at >= ? AND created_at <= ?", timeRange.Start, timeRange.End).
		Pluck("user_id", &userIds).Error

	return userIds, err
}

func (r *UsageRepository) GetUserTotalCostByTimeRange(
	ctx context.Context,
	userId uuid.UUID,
	timeRange models.TimeRange,
) (*models.UsageBilling, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserTotalCostByTimeRange").
			Observe(time.Since(now).Seconds())
	}()

	var usageCost models.UsageBilling
	err := r.db.WithContext(ctx).
		Model(&models.Usage{}).
		Select(`
			COALESCE(SUM(CAST(delivery_cost AS numeric)), 0) as delivery_cost,
			COALESCE(SUM(CAST(transcode_cost AS numeric)), 0) as transcode_cost,
			COALESCE(SUM(CAST(storage_cost AS numeric)), 0) as storage_cost,
			COALESCE(SUM(CAST(total_cost AS numeric)), 0) as total_cost
		`).
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userId, timeRange.Start, timeRange.End).
		Scan(&usageCost).Error

	return &usageCost, err
}

func (r *UsageRepository) GetUserDebt(
	ctx context.Context,
	userId uuid.UUID,
) (decimal.Decimal, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserDebt").
			Observe(time.Since(now).Seconds())
	}()

	var debt decimal.Decimal
	if err := r.db.WithContext(ctx).
		Model(&models.Usage{}).
		Select("COALESCE(SUM(CAST(total_cost AS numeric)), 0) as debt").
		Where("user_id = ? and status = ?", userId, models.FailStatus).
		Scan(&debt).Error; err != nil {
		return decimal.Zero, err
	}

	return debt, nil
}

func (r *UsageRepository) GetUserTocalCostByDate(
	ctx context.Context,
	userId uuid.UUID,
	cursor time.Time,
) (decimal.Decimal, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserTocalCostByDate").
			Observe(time.Since(now).Seconds())
	}()

	var totalCost decimal.Decimal
	if err := r.db.WithContext(ctx).
		Model(&models.Usage{}).
		Select("COALESCE(SUM(CAST(total_cost AS numeric)), 0) as total_cost").
		Where("user_id = ? AND date_trunc('day', created_at) = ?", userId, cursor).
		Scan(&totalCost).Error; err != nil {
		return decimal.Zero, err
	}

	return decimal.Zero, nil
}
