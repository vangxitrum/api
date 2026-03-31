package services

import (
	"context"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
)

type MailService struct {
	emailConnectionRepo models.EmailConnectionRepository
	mailRepo            models.MailRepository
}

func NewMailService(
	emailConnectionRepo models.EmailConnectionRepository,
	mailRepo models.MailRepository,
) *MailService {
	return &MailService{
		emailConnectionRepo: emailConnectionRepo,
		mailRepo:            mailRepo,
	}
}

func (s *MailService) DeleteExpiredEmailType(ctx context.Context) error {
	if err := s.mailRepo.DeleteExpiredMailType(ctx); err != nil {
		return err
	}
	return nil
}
