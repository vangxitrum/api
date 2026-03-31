package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type mailRepository struct {
	db *gorm.DB
}

func MustNewMailRepository(
	db *gorm.DB, init bool,
) models.MailRepository {
	if init {
		err := db.AutoMigrate(&models.Mail{})
		if err != nil {
			panic(err)
		}
	}

	return &mailRepository{
		db: db,
	}
}

func (r *mailRepository) CreateMail(
	ctx context.Context, email *models.Mail,
) error {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateMail").Observe(time.Since(t).Seconds())
	}()

	timeout, cancel := context.WithTimeout(
		ctx,
		2*time.Second,
	)
	defer cancel()

	result := r.db.WithContext(timeout).Create(&email)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *mailRepository) GetMailByUserMail(
	ctx context.Context, email string,
	mailType string,
) (*models.Mail, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMailByUserMail").
			Observe(time.Since(t).Seconds())
	}()

	timeout, cancel := context.WithTimeout(
		ctx,
		2*time.Second,
	)
	defer cancel()

	var rs models.Mail

	if err := r.db.WithContext(ctx).WithContext(timeout).Model(models.Mail{}).Where(
		"mail = ? and type = ? and expired_at >= ?",
		email,
		mailType,
		time.Now().UTC(),
	).First(&rs).Error; err != nil {
		return nil, err
	}

	return &rs, nil
}

func (r *mailRepository) UpdateMail(
	ctx context.Context, email *models.Mail,
) error {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateMail").Observe(time.Since(t).Seconds())
	}()

	timeout, cancel := context.WithTimeout(
		ctx,
		2*time.Second,
	)
	defer cancel()

	result := r.db.WithContext(timeout).Model(
		models.Mail{},
	).Where(
		"id = ?",
		email.Id,
	).Save(&email)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *mailRepository) DeleteExpiredMailType(
	ctx context.Context,
) error {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteExpiredMailType").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).Model(
		models.Mail{},
	).Where(
		"expired_at < ? and status = ?",
		time.Now().UTC(),
		models.DoneStatus,
	).Delete(&models.Mail{})

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *mailRepository) DeleteMailByUserMail(
	ctx context.Context,
	userMail string,
	mailType string,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteMailByUserId").
			Observe(time.Since(now).Seconds())
	}()
	if err := r.db.WithContext(ctx).Delete(
		&models.Mail{},
		"mail = ? AND type = ?",
		userMail,
		mailType,
	).Error; err != nil {
		return err
	}
	return nil
}

func (r *mailRepository) GetMailsByType(
	ctx context.Context,
	mailType string,
) ([]*models.Mail, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMailsByType").
			Observe(time.Since(now).Seconds())
	}()

	var rs []*models.Mail

	if err := r.db.WithContext(ctx).Model(
		models.Mail{},
	).Where(
		"type = ?",
		mailType,
	).Find(&rs).Error; err != nil {
		return nil, err
	}

	return rs, nil
}
