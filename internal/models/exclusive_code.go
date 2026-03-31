package models

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/random"
)

type ExclusiveCodeRepository interface {
	Create(context.Context, *ExclusiveCode) error
	CreateRequestJoinExclusiveProgram(context.Context, *JoinExclusiveProgramRequest) error

	GetExclusiveCodeByCode(ctx context.Context, code string) (*ExclusiveCode, error)
	GetJoinRequestByEmail(ctx context.Context, email string) (*JoinExclusiveProgramRequest, error)

	GenerateExclusiveCodes(ctx context.Context) error

	UpdateExclusiveCodeStatus(ctx context.Context, code string, status string) error
	UpdateJoinRequest(ctx context.Context, req *JoinExclusiveProgramRequest) error

	Delete(ctx context.Context, code string) error
}

type ExclusiveCode struct {
	Code      string          `json:"code"       gorm:"primaryKey"`
	Amount    decimal.Decimal `json:"amount"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at" gorm:"autoCreateTime"`
}

func NewExclusiveCode(amount decimal.Decimal) *ExclusiveCode {
	return &ExclusiveCode{
		Code:      random.GenerateRandomString(8),
		Amount:    amount,
		Status:    ActiveStatus,
		CreatedAt: time.Now(),
	}
}
