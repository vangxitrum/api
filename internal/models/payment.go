package models

import (
	"context"
	"time"

	payment_gateway_model "github.com/AIOZNetwork/payment/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PaymentRepository interface {
	CreatePaymentLog(context.Context, *PaymentLog) error

	GetUserTopUps(context.Context, uuid.UUID, GetPaymentInput) ([]*TopUp, int64, error)
	GetUserBillings(context.Context, uuid.UUID, GetPaymentInput) ([]*Billing, int64, error)
	GetTxOutById(context.Context, uuid.UUID) (*payment_gateway_model.TxOut, error)

	GetTotalUsageByInterval(context.Context, uuid.UUID, time.Time, time.Time) (*Billing, error)
}

var (
	PaymentTypeTopUp  = "top_up"
	PaymentTypeBill   = "bill"
	PaymentTypeRefund = "refund"
)

func NewPaymentLog(
	amount, balance decimal.Decimal,
	paymentType string,
	userId uuid.UUID,
	belongsToId uuid.UUID,
) *PaymentLog {
	if paymentType == PaymentTypeBill {
		amount = amount.Neg()
	}

	return &PaymentLog{
		Id:          uuid.New(),
		Amount:      amount,
		Balance:     balance,
		Type:        paymentType,
		UserId:      userId,
		BelongsToId: belongsToId,
		CreatedAt:   time.Now().UTC(),
	}
}

type PaymentLog struct {
	Id          uuid.UUID       `json:"id"              gorm:"primaryKey;type:uuid"`
	Balance     decimal.Decimal `json:"current_balance"`
	Amount      decimal.Decimal `json:"amount"`
	Type        string          `json:"type"`
	UserId      uuid.UUID       `json:"user_id"         gorm:"type:uuid"`
	BelongsToId uuid.UUID       `json:"belongs_to_id"`
	CreatedAt   time.Time       `json:"created_at"`
}

type TopUp struct {
	TransactionId string          `json:"transaction_id"`
	From          string          `json:"from"`
	Status        string          `json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
	Amount        float64         `json:"float64"`
	Credit        decimal.Decimal `json:"credit"`
}

type Billing struct {
	Storage       int64           `json:"storage"`
	Transcode     float64         `json:"transcode"`
	Delivery      int64           `json:"delivery"`
	StorageCost   decimal.Decimal `json:"storage_cost"`
	TranscodeCost decimal.Decimal `json:"transcode_cost"`
	DeliveryCost  decimal.Decimal `json:"delivery_cost"`
	CreatedAt     time.Time       `json:"created_at"`
}

type GetPaymentInput struct {
	Offset   int
	Limit    int
	OrderBy  string
	From, To time.Time
}
