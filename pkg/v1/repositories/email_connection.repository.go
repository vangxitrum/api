package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type EmailConnectionRepository struct {
	db *gorm.DB
}

func MustNewEmailConnectionRepository(
	db *gorm.DB, init bool,
) models.EmailConnectionRepository {
	if init {
		if err := db.AutoMigrate(&models.EmailConnection{}); err != nil {
			panic("failed to migrate email connection model")
		}
	}

	return &EmailConnectionRepository{
		db: db,
	}
}

func (r *EmailConnectionRepository) Create(
	ctx context.Context,
	emailConnection *models.EmailConnection,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetFormatByMediaId").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(emailConnection).Error
}

func (r *EmailConnectionRepository) GetEmailConnectionByEmail(
	ctx context.Context,
	email string,
) (*models.EmailConnection, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetEmailConnectionByEmail").
			Observe(time.Since(t).Seconds())
	}()

	var emailConnection models.EmailConnection
	if err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&emailConnection).Error; err != nil {
		return nil, err
	}

	return &emailConnection, nil
}

func (r *EmailConnectionRepository) UpdateEmailConnection(
	ctx context.Context,
	emailConnection *models.EmailConnection,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateEmailConnection").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Save(emailConnection).Error
}

func (r *EmailConnectionRepository) UpdateRetries(
	ctx context.Context,
	email string,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateRetries").
			Observe(time.Since(t).Seconds())
	}()
	if err := r.db.WithContext(ctx).Model(&models.EmailConnection{}).Where("email = ?", email).Update("max_retries", gorm.Expr("max_retries - ?", 1)).Error; err != nil {
		return err
	}

	return nil
}
