package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type IpInfoRepository struct {
	db *gorm.DB
}

func MustNewIpInfoRepository(db *gorm.DB, init bool) models.IpInfoRepository {
	if init {
		if err := db.AutoMigrate(&models.IpInfo{}); err != nil {
			panic(err)
		}
	}

	return &IpInfoRepository{
		db: db,
	}
}

func (r *IpInfoRepository) Save(ctx context.Context, ip *models.IpInfo) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("SaveIpInfo").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Save(ip).Error
}

func (r *IpInfoRepository) GetIpInfo(ctx context.Context, ip string) (*models.IpInfo, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetIpInfo").
			Observe(time.Since(t).Seconds())
	}()

	var ipInfo models.IpInfo
	err := r.db.WithContext(ctx).
		Where("ip = ?", ip).
		First(&ipInfo).Error
	return &ipInfo, err
}
