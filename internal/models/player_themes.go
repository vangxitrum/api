package models

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PlayerThemeRepository interface {
	CreatePlayerTheme(
		context.Context, *PlayerTheme,
	) (*PlayerTheme, error)
	GetUserPlayerThemeById(
		context.Context, uuid.UUID, uuid.UUID,
	) (*PlayerTheme, error)
	GetPlayerThemeById(
		context.Context, uuid.UUID,
	) (*PlayerTheme, error)

	DeletePlayerThemeById(
		context.Context, uuid.UUID, uuid.UUID,
	) error
	UpdatePlayerThemeById(
		context.Context, uuid.UUID,
		*PlayerTheme,
	) (*PlayerTheme, error)
	GetPlayerThemeList(
		context.Context,
		GetThemePlayerList,
	) ([]*PlayerTheme, int64, error)

	UpdatePlayerThemeAsset(
		context.Context, uuid.UUID, Asset,
	) (*PlayerTheme, error)

	DeletePlayerThemeAsset(
		context.Context, uuid.UUID, uuid.UUID,
	) error

	AddPlayerThemeToMediaById(
		context.Context, uuid.UUID, uuid.UUID,
	) error
	RemovePlayerThemeFromMedia(
		context.Context, uuid.UUID, uuid.UUID, uuid.UUID,
	) error

	GetActiveMediaByPlayerThemeId(context.Context, uuid.UUID, uuid.UUID) (*Media, error)
	GetDefaultPlayerTheme(context.Context, uuid.UUID) (*PlayerTheme, error)
	UpdateDefaultPlayerTheme(context.Context, uuid.UUID, uuid.UUID) error
}

type PlayerTheme struct {
	Id        uuid.UUID `json:"id"              gorm:"primaryKey;type:uuid"`
	UserId    uuid.UUID `json:"user_id"         gorm:"type:uuid"`
	Name      string    `json:"name"`
	Theme     Theme     `json:"theme"           gorm:"embedded;embeddedPrefix:player_theme_"`
	Controls  Controls  `json:"controls"        gorm:"embedded;embeddedPrefix:player_controls_"`
	Asset     Asset     `json:"asset,omitempty" gorm:"embedded;embeddedPrefix:player_asset_"`
	CreatedAt time.Time `json:"created_at"`
	IsDefault *bool     `json:"is_default"`
} //	@name	PlayerTheme

type Theme struct {
	MainColor                 string `json:"main_color"`
	TextColor                 string `json:"text_color"`
	TextTrackColor            string `json:"text_track_color"`
	TextTrackBackground       string `json:"text_track_background"`
	ControlBarHeight          string `json:"control_bar_height"`
	ControlBarBackgroundColor string `json:"control_bar_background_color"`
	ProgressBarHeight         string `json:"progress_bar_height"`
	ProgressBarCircleSize     string `json:"progress_bar_circle_size"`
	MenuBackGroundColor       string `json:"menu_background_color"`
	MenuItemBackGroundHover   string `json:"menu_item_background_hover"`
} //	@name	Theme

type Controls struct {
	EnableAPI      *bool `json:"enable_api"`
	EnableControls *bool `json:"enable_controls"`
	ForceAutoplay  *bool `json:"force_autoplay"`
	HideTitle      *bool `json:"hide_title"`
	ForceLoop      *bool `json:"force_loop"`
} //	@name	Controls

type Asset struct {
	FileId        *string  `json:"-"`
	File          *CdnFile `json:"-"                         gorm:"foreignKey:FileId;references:Id"`
	LogoImageLink string   `json:"logo_image_link,omitempty" gorm:"-"`
	LogoLink      string   `json:"logo_link,omitempty"`
} //	@name	Asset

func NewPlayerTheme(
	userId uuid.UUID,
	name string,
	Theme Theme,
) *PlayerTheme {
	defaultControl := true
	isDefault := false
	return &PlayerTheme{
		Id:     uuid.New(),
		UserId: userId,
		Name:   name,
		Theme:  Theme,
		Controls: Controls{
			EnableAPI:      &defaultControl,
			EnableControls: &defaultControl,
			ForceAutoplay:  &defaultControl,
			HideTitle:      &defaultControl,
			ForceLoop:      &defaultControl,
		},
		IsDefault: &isDefault,
		CreatedAt: time.Now(),
	}
}

type PlayerThemeInput struct {
	Name      string    `json:"name"`
	Theme     Theme     `json:"theme,omitempty"`
	Controls  *Controls `json:"controls,omitempty"`
	IsDefault *bool     `json:"is_default"`
} //	@name	PlayerThemeInput

type GetThemePlayerList struct {
	UserId uuid.UUID
	SortBy string
	Order  string
	Offset uint64
	Limit  uint64
	Search string
}
