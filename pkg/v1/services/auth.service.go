package services

import "10.0.0.50/tuan.quang.tran/vms-v2/internal/models"

type AuthService struct {
	userRepo models.UserRepository
}

func NewAuthService(
	userRepo models.UserRepository,
) *AuthService {
	return &AuthService{
		userRepo: userRepo,
	}
}
