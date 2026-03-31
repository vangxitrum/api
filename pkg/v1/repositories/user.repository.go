package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type UserRepository struct {
	db *gorm.DB
}

func MustNewUserRepository(
	db *gorm.DB, init bool,
) models.UserRepository {
	if init {
		if err := db.AutoMigrate(&models.User{}, &models.SubscribeInfo{}); err != nil {
			panic(err)
		}
	}

	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) Create(
	ctx context.Context,
	user *models.User,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateUser").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(user).Error
}

func (r *UserRepository) CreateSubscribeInfo(
	ctx context.Context,
	info *models.SubscribeInfo,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateSubcribeInfo").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(info).Error
}

func (r *UserRepository) GetUserById(
	ctx context.Context,
	id uuid.UUID,
) (*models.User, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserById").
			Observe(time.Since(t).Seconds())
	}()

	var user models.User
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) UpdateUsersFreeBalance(
	ctx context.Context,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateFreeBalances").
			Observe(time.Since(t).Seconds())
	}()

	query := `
	 Update wallets
	 Set free_balance = 0
	 Where redeem_at < now() - interval '1 month' and free_balance > 0
	`

	if err := r.db.WithContext(ctx).
		Exec(query).Error; err != nil {
		return err
	}

	return nil
}

func (r *UserRepository) GetActiveUserByEmail(
	ctx context.Context,
	email string,
) (*models.User, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserByEmail").
			Observe(time.Since(t).Seconds())
	}()

	var user models.User
	if err := r.db.WithContext(ctx).
		Where("email = ? and deleted_at is null", email).
		First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetUserByWalletAddress(
	ctx context.Context,
	walletAddress string,
) (*models.User, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserByWalletAddress").
			Observe(time.Since(t).Seconds())
	}()

	var user models.User
	if err := r.db.WithContext(ctx).
		Where(
			"wallet_address = ?",
			walletAddress,
		).
		First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetActiveUserByWalletConnection(
	ctx context.Context,
	walletConnection string,
) (*models.User, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserByWalletConnection").
			Observe(time.Since(t).Seconds())
	}()

	var user models.User
	if err := r.db.WithContext(ctx).
		Where(
			"wallet_connection = ? and deleted_at is null",
			walletConnection,
		).
		First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetActiveAllUsers(
	ctx context.Context,
) ([]*models.User, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetAllUsers").
			Observe(time.Since(t).Seconds())
	}()

	var users []*models.User
	if err := r.db.WithContext(ctx).
		Where("deleted_at is null").
		Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) GetDeletingUsers(
	ctx context.Context,
) ([]*models.User, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetDeletingUsers").
			Observe(time.Since(t).Seconds())
	}()

	var users []*models.User
	if err := r.db.WithContext(ctx).
		Where("status != ? and deleted_at is not null", models.DeletedStatus).
		Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) GetUserByEmailAndStatus(
	ctx context.Context,
	email string,
	status string,
) (*models.User, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserByEmailAndStatus").
			Observe(time.Since(t).Seconds())
	}()

	var user models.User
	if err := r.db.WithContext(ctx).
		Where("email = ? AND status = ?", email, status).
		First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetUsersByStatus(
	ctx context.Context,
	status string,
) ([]*models.User, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUsersByStatus").
			Observe(time.Since(t).Seconds())
	}()

	var users []*models.User
	if err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) UpdateUser(
	ctx context.Context,
	user *models.User,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateUser").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Save(user).Error
}

func (r *UserRepository) UpdateUserPriceInfo(
	ctx context.Context,
	userId uuid.UUID,
	price float64,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateUserPriceInfo").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userId).
		Updates(map[string]any{
			"aioz_price":            price,
			"last_price_updated_at": time.Now().UTC(),
			"updated_at":            time.Now().UTC(),
		}).Error
}

func (r *UserRepository) UpdateUserInfo(
	ctx context.Context,
	userId uuid.UUID,
	firstName, lastName string,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateUserInfo").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userId).
		Updates(map[string]any{
			"first_name": firstName,
			"last_name":  lastName,
			"updated_at": time.Now().UTC(),
		}).Error
}

func (r *UserRepository) UpdateUserStatus(
	ctx context.Context,
	userId uuid.UUID,
	status string,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateUserStatus").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userId).
		Updates(map[string]any{
			"status":     status,
			"updated_at": time.Now().UTC(),
		}).Error
}

func (r *UserRepository) UpdateUserMediaConfig(
	ctx context.Context,
	userId uuid.UUID,
	mediaConfig string,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateUserMediaConfig").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userId).
		Updates(map[string]any{
			"media_qualities_config": mediaConfig,
			"updated_at":             time.Now().UTC(),
		}).Error
}

func (r *UserRepository) UpdateUserLastRequestedAt(
	ctx context.Context,
	userId uuid.UUID,
	lastRequestedAt time.Time,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateUserLastRequestedAt").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userId).
		Updates(map[string]any{
			"last_requested_at": lastRequestedAt,
		}).Error
}
