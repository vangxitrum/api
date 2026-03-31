package models

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Mail struct {
	Id        uuid.UUID `json:"id"         gorm:"type:uuid;primaryKey"`
	Mail      string    `json:"mail"       gorm:"index"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiredAt time.Time `json:"expired_at"`
	Type      string    `json:"type"`
}

type MailRepository interface {
	CreateMail(context.Context, *Mail) error

	GetMailByUserMail(
		context.Context, string, string,
	) (*Mail, error)
	GetMailsByType(context.Context, string) ([]*Mail, error)

	UpdateMail(context.Context, *Mail) error

	DeleteMailByUserMail(context.Context, string, string) error
	DeleteExpiredMailType(
		context.Context,
	) error
}

func NewMail(
	mail, mailType string,
	CreatedAt, ExpiredAt time.Time,
) *Mail {
	return &Mail{
		Id:        uuid.New(),
		Mail:      mail,
		Status:    DoneStatus,
		Type:      mailType,
		CreatedAt: CreatedAt,
		ExpiredAt: ExpiredAt,
	}
}
