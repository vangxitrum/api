package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type WalletConnectionRepository struct {
	db *gorm.DB
}

func MustNewWalletConnectionRepository(
	db *gorm.DB, init bool,
) models.WalletConnectionRepository {
	if init {
		if err := db.AutoMigrate(&models.WalletConnection{}); err != nil {
			panic("failed to migrate email connection model")
		}
	}

	return &WalletConnectionRepository{
		db: db,
	}
}

func (r *WalletConnectionRepository) Create(
	ctx context.Context,
	walletConnection *models.WalletConnection,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetFormatByMediaId").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(walletConnection).Error
}

func (r *WalletConnectionRepository) GetWalletConnectionByWalletAddress(
	ctx context.Context,
	walletAddress string,
) (*models.WalletConnection, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetWalletConnectionByWalletAddress").
			Observe(time.Since(t).Seconds())
	}()

	var walletConnection models.WalletConnection
	if err := r.db.WithContext(ctx).
		Where("address = ?", walletAddress).
		First(&walletConnection).Error; err != nil {
		return nil, err
	}

	return &walletConnection, nil
}

func (r *WalletConnectionRepository) UpdateWalletConnection(
	ctx context.Context,
	walletConnection *models.WalletConnection,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateWalletConnection").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Save(walletConnection).Error
}

func (r *WalletConnectionRepository) UpdateUserWalletConnection(
	ctx context.Context, userId uuid.UUID,
	address string,
) error {
	result := r.db.WithContext(ctx).Model(&models.User{}).Where(
		"id = ?",
		userId,
	).Updates(
		map[string]interface{}{
			"wallet_connection": address,
			"updated_at":        time.Now().UTC(),
		},
	)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
