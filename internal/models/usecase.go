package models

type UseCase struct {
	userRepository    UserRepository
	apiKeyRepository  ApiKeyRepository
	webhookRepository WebhookRepository
}

func NewUseCase(
	userRepository UserRepository,
	apiKeyRepository ApiKeyRepository,
	webhookRepository WebhookRepository,
) UseCase {
	return UseCase{
		userRepository:    userRepository,
		apiKeyRepository:  apiKeyRepository,
		webhookRepository: webhookRepository,
	}
}

func (u *UseCase) UserRepository() UserRepository {
	return u.userRepository
}

func (u *UseCase) ApiKeyRepository() ApiKeyRepository {
	return u.apiKeyRepository
}

func (u *UseCase) WebhookRepository() WebhookRepository {
	return u.webhookRepository
}
