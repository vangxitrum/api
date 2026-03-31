package repositories

import (
	"context"
	"math"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type ExclusiveCodeRepository struct {
	db *gorm.DB
}

func MustNewExclusiveCodeRepository(db *gorm.DB, init bool) models.ExclusiveCodeRepository {
	if init {
		if err := db.AutoMigrate(&models.ExclusiveCode{}, &models.JoinExclusiveProgramRequest{}); err != nil {
			panic("failed to auto migrate ExclusiveCode: " + err.Error())
		}
	}

	return &ExclusiveCodeRepository{db: db}
}

func (r *ExclusiveCodeRepository) Create(ctx context.Context, code *models.ExclusiveCode) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateApiKey").Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).Create(code).Error
}

func (r *ExclusiveCodeRepository) CreateRequestJoinExclusiveProgram(
	ctx context.Context,
	req *models.JoinExclusiveProgramRequest,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateRequestJoinExclusiveProgram").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).Create(req).Error
}

func (r *ExclusiveCodeRepository) GenerateExclusiveCodes(
	ctx context.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GenerateExclusiveCode").
			Observe(time.Since(t).Seconds())
	}()

	var total int64
	if err := r.db.WithContext(ctx).
		Model(&models.ExclusiveCode{}).
		Count(&total).Error; err != nil {
		return err
	}

	if total == 0 {
		for range 1000 {
			code := models.NewExclusiveCode(
				decimal.NewFromInt(20).Mul(decimal.NewFromFloat(math.Pow10(18))),
			)

			if err := r.Create(ctx, code); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *ExclusiveCodeRepository) GetExclusiveCodeByCode(
	ctx context.Context,
	code string,
) (*models.ExclusiveCode, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetExclusiveCodeByCode").
			Observe(time.Since(t).Seconds())
	}()

	var exclusiveCode models.ExclusiveCode
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&exclusiveCode).Error
	if err != nil {
		return nil, err
	}

	return &exclusiveCode, nil
}

func (r *ExclusiveCodeRepository) GetJoinRequestByEmail(
	ctx context.Context,
	email string,
) (*models.JoinExclusiveProgramRequest, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetJoinRequestByEmail").
			Observe(time.Since(t).Seconds())
	}()

	var req models.JoinExclusiveProgramRequest
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&req).Error
	if err != nil {
		return nil, err
	}

	return &req, nil
}

func (r *ExclusiveCodeRepository) UpdateExclusiveCodeStatus(
	ctx context.Context,
	code string,
	status string,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateExclusiveCode").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).Model(&models.ExclusiveCode{}).
		Where("code = ?", code).
		Update("status", status).Error
}

func (r *ExclusiveCodeRepository) UpdateJoinRequest(
	ctx context.Context,
	req *models.JoinExclusiveProgramRequest,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateJoinRequest").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).Save(req).Error
}

func (r *ExclusiveCodeRepository) Delete(ctx context.Context, code string) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteExclusiveCode").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).Where("code = ?", code).Delete(&models.ExclusiveCode{}).Error
}
