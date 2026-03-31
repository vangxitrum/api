package repositories

import (
	"context"
	"fmt"
	"time"

	payment_gateway_model "github.com/AIOZNetwork/payment/models"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type PaymentRepository struct {
	db *gorm.DB
}

func MustNewPaymentRepository(db *gorm.DB, init bool) models.PaymentRepository {
	if init {
		if err := db.AutoMigrate(&models.PaymentLog{}); err != nil {
			panic("failed to migrate payment models")
		}
	}

	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) CreatePaymentLog(
	ctx context.Context,
	paymentLog *models.PaymentLog,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreatePaymentLog").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(paymentLog).Error
}

func (r *PaymentRepository) GetUserTopUps(
	ctx context.Context,
	userId uuid.UUID,
	input models.GetPaymentInput,
) ([]*models.TopUp, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserTopUps").
			Observe(time.Since(t).Seconds())
	}()

	var topUps []*models.TopUp
	var total int64

	countQuery := `
		SELECT coalesce(count(*),0) as total
		FROM tx_outs t
		JOIN users u on t.sender = u.wallet_address and type_tx = 'normal'
		WHERE u.id = ? and t.created_at >= ? and t.created_at <= ?
	`
	query := `
		SELECT coalesce(t.cosmos_tx_in_hash,t.evm_tx_in_hash) as transaction_id
			, sender as "from"
			, case when credit != 0  then 'success' else 'pending' end as status
			, t.created_at
			, t.amount
			, t.credit
		FROM tx_outs t
		JOIN users u on t.sender = u.wallet_address and type_tx = 'normal'
		WHERE u.id = ? and t.created_at >= ? and t.created_at <= ?
	`

	if err := r.db.Raw(
		countQuery,
		userId,
		input.From.Format(time.RFC3339),
		input.To.Format(time.RFC3339),
	).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*models.TopUp{}, 0, nil
	}

	if input.OrderBy != "" {
		query += " ORDER BY  t.created_at " + input.OrderBy
	} else {
		query += " ORDER BY t.created_at DESC"
	}

	if input.Offset != 0 {
		query += fmt.Sprintf(" OFFSET %d", input.Offset)
	}

	if input.Limit != 0 {
		query += fmt.Sprintf(" LIMIT %d", input.Limit)
	}

	if err := r.db.Raw(
		query,
		userId,
		input.From.Format(time.RFC3339),
		input.To.Format(time.RFC3339),
	).Scan(&topUps).Error; err != nil {
		return nil, 0, err
	}
	return topUps, total, nil
}

func (r *PaymentRepository) GetTxOutById(
	ctx context.Context,
	id uuid.UUID,
) (*payment_gateway_model.TxOut, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetTxOutById").
			Observe(time.Since(t).Seconds())
	}()

	var txOut payment_gateway_model.TxOut
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&txOut).Error; err != nil {
		return nil, err
	}

	return &txOut, nil
}

func (r *PaymentRepository) GetUserBillings(
	ctx context.Context,
	userId uuid.UUID,
	input models.GetPaymentInput,
) ([]*models.Billing, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserBillings").
			Observe(time.Since(t).Seconds())
	}()

	var billings []*models.Billing
	var total int64

	countQuery := `
	select count(*) as total
	from (
		SELECT user_id,
			coalesce(max(storage),0) as storage,
			coalesce(sum(transcode),0) as transcode,
			coalesce(sum(delivery),0) as delivery,
			coalesce(sum(cast(storage_cost as numeric)),0) as storage_cost,
			coalesce(sum(cast(transcode_cost as numeric)),0) as transcode_cost,
			coalesce(sum(cast(delivery_cost as numeric)),0) as delivery_cost,
			DATE_TRUNC('day',created_at at time zone 'utc') as created_at
		FROM usages
		WHERE user_id = ?
		GROUP BY DATE_TRUNC('day',created_at at time zone 'utc'),user_id
	)
	WHERE (storage > 0 OR transcode > 0 OR delivery > 0)
	AND created_at >= ? AND created_at <= ?
	`

	query := fmt.Sprintf(`
		SELECT *
		FROM (
				SELECT user_id,
					coalesce(max(storage),0) as storage,
					coalesce(sum(transcode),0) as transcode,
					coalesce(sum(delivery),0) as delivery,
					coalesce(sum(cast(storage_cost as numeric)),0) as storage_cost,
					coalesce(sum(cast(transcode_cost as numeric)),0) as transcode_cost,
					coalesce(sum(cast(delivery_cost as numeric)),0) as delivery_cost,
					DATE_TRUNC('day',created_at at time zone 'utc') as created_at
				FROM usages
				WHERE user_id = ?
				GROUP BY DATE_TRUNC('day',created_at at time zone 'utc'),user_id
				ORDER BY DATE_TRUNC('day',created_at at time zone 'utc') %s
		) as sub
		WHERE (storage > 0 OR transcode > 0 OR delivery > 0)
			AND created_at >= ? AND created_at <= ?

		OFFSET %d
		LIMIT %d
	`, input.OrderBy, input.Offset, input.Limit)

	if err := r.db.Raw(
		countQuery,
		userId,
		input.From.Format(time.RFC3339),
		input.To.Format(time.RFC3339),
	).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*models.Billing{}, 0, nil
	}

	if err := r.db.Raw(
		query,
		userId,
		input.From.Format(time.RFC3339),
		input.To.Format(time.RFC3339),
	).Scan(&billings).Error; err != nil {
		return nil, 0, err
	}

	if len(billings) == 0 {
		return []*models.Billing{}, total, nil
	}

	return billings, total, nil
}

func (r *PaymentRepository) GetTotalUsageByInterval(
	ctx context.Context,
	userId uuid.UUID,
	start time.Time,
	end time.Time,
) (*models.Billing, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetTotalUsageByInterval").
			Observe(time.Since(t).Seconds())
	}()

	var result *models.Billing
	query := `SELECT user_id,
			coalesce(max(storage),0) as storage,
			coalesce(sum(transcode),0) as transcode,
			coalesce(sum(delivery),0) as delivery,
			coalesce(sum(cast(storage_cost as numeric)),0) as storage_cost,
			coalesce(sum(cast(transcode_cost as numeric)),0) as transcode_cost,
			coalesce(sum(cast(delivery_cost as numeric)),0) as delivery_cost
		FROM usages
		WHERE user_id = ? AND created_at BETWEEN ? AND ?
		GROUP BY user_id
	`
	if err := r.db.Raw(query, userId, start, end).Scan(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}
