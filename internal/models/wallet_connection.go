package models

import (
	"time"

	"github.com/google/uuid"

	"golang.org/x/net/context"
)

type WalletConnectionRepository interface {
	Create(
		context.Context, *WalletConnection,
	) error

	GetWalletConnectionByWalletAddress(
		context.Context, string,
	) (*WalletConnection, error)

	UpdateWalletConnection(
		context.Context, *WalletConnection,
	) error

	UpdateUserWalletConnection(
		context.Context, uuid.UUID,
		string,
	) error
}

type WalletConnection struct {
	Address   string    `json:"address" gorm:"primaryKey"`
	Challenge string    `json:"challenge"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewWalletConnection(address, challenge string) *WalletConnection {
	return &WalletConnection{
		Address:   address,
		Challenge: challenge,
		CreatedAt: time.Now(),
	}
}
