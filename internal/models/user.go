package models

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type UserRepository interface {
	Create(context.Context, *User) error
	CreateSubscribeInfo(context.Context, *SubscribeInfo) error

	GetUserById(context.Context, uuid.UUID) (*User, error)
	GetActiveUserByEmail(context.Context, string) (*User, error)
	GetUserByEmailAndStatus(context.Context, string, string) (*User, error)
	GetUserByWalletAddress(context.Context, string) (*User, error)
	GetActiveUserByWalletConnection(context.Context, string) (*User, error)
	GetActiveAllUsers(context.Context) ([]*User, error)
	GetUsersByStatus(context.Context, string) ([]*User, error)
	GetDeletingUsers(context.Context) ([]*User, error)

	UpdateUser(context.Context, *User) error
	UpdateUserPriceInfo(context.Context, uuid.UUID, float64) error
	UpdateUserInfo(context.Context, uuid.UUID, string, string) error
	UpdateUserStatus(context.Context, uuid.UUID, string) error
	UpdateUserMediaConfig(context.Context, uuid.UUID, string) error
	UpdateUsersFreeBalance(context.Context) error
	UpdateUserLastRequestedAt(context.Context, uuid.UUID, time.Time) error
}

type User struct {
	Id                   uuid.UUID       `json:"id"                          gorm:"primaryKey;type:uuid"`
	FirstName            string          `json:"first_name"`
	LastName             string          `json:"last_name"`
	FullName             string          `json:"-"                           gorm:"-"`
	Email                string          `json:"email"`
	WalletAddress        string          `json:"wallet_address"`
	Debt                 decimal.Decimal `json:"debt,omitempty"              gorm:"-"`
	Balance              decimal.Decimal `json:"balance"                     gorm:"-"`
	WalletConnection     string          `json:"wallet_connection"`
	MediaQualitiesConfig string          `json:"media_qualities_config"`
	AiozPrice            float64         `json:"-"`
	LastPriceUpdatedAt   time.Time       `json:"-"`
	Status               string          `json:"-"`
	ExclusiveCode        string          `json:"exclusive_code,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	DeletedAt            *time.Time      `json:"deleted_at,omitempty"`
	LastRequestedAt      *time.Time      `json:"last_requested_at,omitempty"`
} //	@name	User

func NewUser(email, firstName, lastName string) *User {
	return &User{
		Id:                   uuid.New(),
		FirstName:            firstName,
		LastName:             lastName,
		Email:                email,
		Status:               ActiveStatus,
		MediaQualitiesConfig: strings.Join(DefaultMediaQualities, ","),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}

func NewUserWithWallet(walletConnection string) *User {
	return &User{
		Id:               uuid.New(),
		WalletConnection: walletConnection,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func (u *User) LogValue() slog.Value {
	return slog.StringValue("user_id:" + u.Id.String())
}

func (u *User) IsDeleted() bool {
	return u.Status == DeletedStatus || u.DeletedAt != nil
}

type JoinExclusiveProgramInput struct {
	OrganizationName      string
	Email                 string
	Role                  string
	ContentType           string
	StorageUsage          float32
	DeliveryUsage         float32
	StreamPlatforms       string
	ReasonUsingAIOZStream string
}
