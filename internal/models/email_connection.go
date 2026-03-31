package models

import (
	"context"
	"time"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/random"
)

type EmailConnection struct {
	Email      string    `json:"email"       gorm:"primaryKey;Index"`
	Code       string    `json:"login_code"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	MaxRetries int       `json:"max_retries"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ExpiredAt  time.Time `json:"expired_at"  gorm:"index"`
}

type EmailConnectionRepository interface {
	Create(
		context.Context, *EmailConnection,
	) error

	GetEmailConnectionByEmail(
		context.Context, string,
	) (*EmailConnection, error)

	UpdateEmailConnection(
		context.Context, *EmailConnection,
	) error

	UpdateRetries(
		context.Context, string,
	) error
}

func NewEmailConnection(email string, maxRetries int) *EmailConnection {
	return &EmailConnection{
		Email:      email,
		Code:       random.GenerateRandomCode(6),
		MaxRetries: maxRetries,
		CreatedAt:  time.Now().UTC(),
		ExpiredAt:  time.Now().UTC().Add(ExpiredEmailTime),
	}
}

type EmailConnectionInput struct {
	Email           string
	LoginCode       string
	ExpiresLoginAt  time.Time
	VerifyCode      string
	ExpiresVerifyAt time.Time
	FirstName       string
	LastName        string
}
