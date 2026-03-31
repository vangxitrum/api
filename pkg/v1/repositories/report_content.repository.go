package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type ReportContentRepository struct {
	db *gorm.DB
}

func MustNewReportContentRepository(db *gorm.DB, init bool) models.ReportContentRepository {
	if init {
		if err := db.AutoMigrate(&models.ContentReport{}); err != nil {
			panic("failed to auto migrate report content models")
		}
	}
	return &ReportContentRepository{db: db}
}

func (r *ReportContentRepository) GetListReports(ctx context.Context, filter models.GetContentReportList) ([]*models.ContentReport, int64, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetListReports").
			Observe(time.Since(t).Seconds())
	}()

	var (
		result []*models.ContentReport
		total  int64
	)

	query := r.db.WithContext(ctx).Model(&models.ContentReport{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Order(filter.SortBy + " " + filter.Order).
		Offset(int(filter.Offset)).
		Limit(int(filter.Limit)).
		Find(&result).Error; err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (r *ReportContentRepository) GetReportByMediaAndIP(ctx context.Context, mediaId uuid.UUID, ipAddress string) (*models.ContentReport, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetReportByMediaAndIP").
			Observe(time.Since(t).Seconds())
	}()

	var result models.ContentReport
	if err := r.db.WithContext(ctx).Model(&models.ContentReport{}).Where("media_id = ? AND reporter_ip = ?", mediaId, ipAddress).First(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *ReportContentRepository) GetTotalReportsByMediaId(ctx context.Context, mediaId uuid.UUID) (int64, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetTotalReportsByMediaId").
			Observe(time.Since(t).Seconds())
	}()

	var total int64
	if err := r.db.WithContext(ctx).Model(&models.ContentReport{}).Where("media_id = ?", mediaId).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *ReportContentRepository) GetReportById(ctx context.Context, id uuid.UUID) (*models.ContentReport, error) {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetReportById").
			Observe(time.Since(t).Seconds())
	}()
	var result models.ContentReport

	if err := r.db.WithContext(ctx).Where(
		&models.ContentReport{
			Id: id,
		},
	).First(&result).Error; err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *ReportContentRepository) CreateReport(ctx context.Context, report *models.ContentReport) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateReport").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).Create(report).Error
}

func (r *ReportContentRepository) UpdateReportStatus(ctx context.Context, id uuid.UUID, status string) error {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateReportStatus").
			Observe(time.Since(t).Seconds())
	}()
	return r.db.WithContext(ctx).Model(&models.ContentReport{}).Where("id = ?", id).Update("status", status).Error
}

func (r *ReportContentRepository) DeleteReport(ctx context.Context, contentReport *models.ContentReport) error {
	t := time.Now()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteReport").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).Delete(contentReport)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil

}
