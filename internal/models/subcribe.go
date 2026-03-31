package models

import (
	"time"
)

type SubscribeInfo struct {
	Email     string    `json:"email"      gorm:"primaryKey;email"`
	CreatedAt time.Time `json:"created_at"`
}

func NewSubcribeInfo(email string) *SubscribeInfo {
	return &SubscribeInfo{
		Email:     email,
		CreatedAt: time.Now(),
	}
}
