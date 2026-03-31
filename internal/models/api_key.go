package models

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mdobak/go-xerrors"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/hash"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/random"
)

type ApiKeyType string

const (
	FullAccess ApiKeyType = "full_access"
	OnlyUpload ApiKeyType = "only_upload"
)

type ApiKey struct {
	Id              uuid.UUID `json:"id"                          gorm:"type:uuid;primaryKey"`
	UserId          uuid.UUID `json:"-"                           gorm:"type:uuid"`
	User            *User     `json:"user,omitempty"              gorm:"foreignKey:UserId;references:Id"`
	Name            string    `json:"name"`
	PublicKey       string    `json:"public_key"`
	Secret          string    `json:"-"`
	Ttl             string    `json:"ttl"`
	Status          string    `json:"status"`
	UnHashedSecret  string    `json:"secret,omitempty"`
	TruncatedSecret string    `json:"truncated_secret"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ExpiredAt       time.Time `json:"expired_at"`
	LastRequestedAt time.Time `json:"last_requested_at,omitempty"`
	Type            string    `json:"type"`
} //	@name	ApiKey

type ApiKeyRepository interface {
	CreateApiKey(
		context.Context, *ApiKey,
	) (*ApiKey, error)

	GetApiKeyList(
		context.Context, GetApiKeyListInput,
	) ([]*ApiKey, int64, error)
	GetApiKeyById(
		context.Context, uuid.UUID,
	) (*ApiKey, error)
	GetApiKeyByUserId(
		context.Context, uuid.UUID,
	) ([]*ApiKey, error)
	GetApiKeyByKey(
		context.Context, string,
	) (*ApiKey, error)
	GetApiKeyListBetweenTime(
		context.Context, uuid.UUID, time.Time,
		time.Time,
	) ([]*ApiKey, error)
	GetUserApiKeyByName(
		context.Context, uuid.UUID, string,
	) (*ApiKey, error)

	DeleteUserApiKeyById(
		context.Context, uuid.UUID, uuid.UUID,
	) error

	UpdateApiKeyName(
		context.Context, *ApiKey,
	) error

	UpdateApiKeyLastRequestedAt(context.Context, uuid.UUID, time.Time) error

	DeleteExpiredApiKey(context.Context) error
	DeleteUserAPIKeys(context.Context, uuid.UUID) error
}

func NewApiKey(
	userId uuid.UUID, name, ttl,
	apiKeyType string,
) (*ApiKey, error) {
	if apiKeyType != string(FullAccess) && apiKeyType != string(OnlyUpload) {
		return nil, xerrors.New("Invalid api key type.")
	}
	newSecret := random.GenerateRandomString(36)
	hashedSecret, err := hash.HashString(newSecret)
	if err != nil {
		return nil, err
	}

	numTtl, err := strconv.ParseInt(ttl, 10, 64)
	if err != nil {
		var numError *strconv.NumError
		if errors.As(err, &numError) {
			if errors.Is(
				numError.Err,
				strconv.ErrRange,
			) {
				return nil, fmt.Errorf("TTL value is out of range for int64.")
			}
		}
		return nil, fmt.Errorf(
			"invalid TTL format: %v",
			err,
		)
	}
	if numTtl < 0 {
		return nil, xerrors.New("TTL value must be greater than or equal to 0.")
	}
	return &ApiKey{
		Id:             uuid.New(),
		UserId:         userId,
		Name:           name,
		Ttl:            ttl,
		Status:         ActiveStatus,
		PublicKey:      random.GenerateRandomString(32),
		Secret:         hashedSecret,
		UnHashedSecret: newSecret,
		TruncatedSecret: random.TruncateString(
			newSecret,
			3,
		),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		ExpiredAt: time.Now().UTC().Add(time.Duration(numTtl) * time.Second),
		Type:      apiKeyType,
	}, nil
}

func (a *ApiKey) IsInvalid() bool {
	return a.Status == DeletedStatus || a.Status == ExpiredStatus
}

var ApiKeysSortByMap = map[string]bool{
	"created_at": true,
	"name":       true,
}

type GetApiKeyListInput struct {
	UserId uuid.UUID
	Search string
	SortBy string
	Order  string
	Offset uint64
	Limit  uint64
	Type   string
}

type DeleteApiKeyInput struct {
	UserId uuid.UUID
	Id     uuid.UUID
}
