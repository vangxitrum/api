package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type PartRepository struct {
	db *gorm.DB
}

func MustNewPartRepository(
	db *gorm.DB,
	init bool,
) models.PartRepository {
	if init {
		if err := db.AutoMigrate(&models.Part{}); err != nil {
			panic(err)
		}
	}

	return &PartRepository{
		db: db,
	}
}

func (r *PartRepository) Create(
	ctx context.Context,
	part *models.Part,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreatePart").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(part).Error
}

func (r *PartRepository) DeletePartsByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeletePartsByMediaId").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Delete(&models.Part{}, "media_id = ?", mediaId).
		Error
}
