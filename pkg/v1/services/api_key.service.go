package services

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
)

type ApiKeyService struct {
	apiKeyRepo models.ApiKeyRepository
}

func NewApiKeyService(
	apiKeyRepo models.ApiKeyRepository,
) *ApiKeyService {
	return &ApiKeyService{
		apiKeyRepo: apiKeyRepo,
	}
}

func (s *ApiKeyService) CreateApiKey(
	ctx context.Context,
	userId uuid.UUID,
	apiKeyName, ttl,
	apiType string,
) (*models.ApiKey, error) {
	newApiKey, err := models.NewApiKey(
		userId,
		apiKeyName,
		ttl,
		apiType,
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if _, err := s.apiKeyRepo.CreateApiKey(
		ctx,
		newApiKey,
	); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return newApiKey, nil
}

func (s *ApiKeyService) GetApiKeyList(
	ctx context.Context,
	params models.GetApiKeyListInput,
) ([]*models.ApiKey, int64, error) {
	result, total, err := s.apiKeyRepo.GetApiKeyList(
		ctx, models.GetApiKeyListInput{
			UserId: params.UserId,
			Search: params.Search,
			SortBy: params.SortBy,
			Order:  params.Order,
			Offset: params.Offset,
			Limit:  params.Limit,
			Type:   params.Type,
		},
	)
	if err != nil {
		return nil, 0, response.NewHttpError(
			http.StatusInternalServerError, err,
			"Failed to get api key list.",
		)
	}

	return result, total, nil
}

func (s *ApiKeyService) ChangeApiKey(
	ctx context.Context,
	apiKeyId, userId uuid.UUID,
	newApiKeyName string,
) error {
	apiKey, err := s.apiKeyRepo.GetApiKeyById(
		ctx,
		apiKeyId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	if apiKey.UserId != userId {
		return response.NewHttpError(
			http.StatusForbidden,
			errors.New("You are not allowed to update this api key."),
		)
	}

	if apiKey.Name == newApiKeyName {
		return nil
	}

	if err := s.apiKeyRepo.UpdateApiKeyName(
		ctx, &models.ApiKey{
			UserId: userId,
			Id:     apiKeyId,
			Name:   newApiKeyName,
		},
	); err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *ApiKeyService) DeleteUserApiKey(
	ctx context.Context, apiKeyId uuid.UUID,
	authInfo models.AuthenticationInfo,
) error {
	if _, err := s.apiKeyRepo.GetApiKeyById(
		ctx,
		apiKeyId,
	); err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	if err := s.apiKeyRepo.DeleteUserApiKeyById(
		ctx,
		apiKeyId,
		authInfo.User.Id,
	); err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}
	return nil
}

func (s *ApiKeyService) DeleteExpiredUserApiKey(ctx context.Context) error {
	if err := s.apiKeyRepo.DeleteExpiredApiKey(ctx); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *ApiKeyService) DeleteUserApiKeys(
	ctx context.Context,
	userId uuid.UUID,
) error {
	if err := s.apiKeyRepo.DeleteUserAPIKeys(ctx, userId); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}
