package models

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PlaylistRepository interface {
	CreatePlaylist(context.Context, *Playlist) (*Playlist, error)
	CreatePlaylistItem(context.Context, *PlaylistItem) (*PlaylistItem, error)
	CreatePlaylistThumbnail(context.Context, *PlaylistThumbnail) error

	GetPlaylistById(context.Context, uuid.UUID) (*Playlist, error)
	GetPlaylistByIds(context.Context, []uuid.UUID) ([]Playlist, error)
	GetPlaylistItemById(context.Context, uuid.UUID) (*PlaylistItem, error)
	GetExistsAnyMediaInPlaylists(context.Context, []uuid.UUID, []uuid.UUID) (bool, error)
	GetPlaylistItemsByPlaylistId(context.Context, uuid.UUID) ([]*PlaylistItem, error)
	GetFirstPlaylistItemByPlaylistId(context.Context, uuid.UUID) (*PlaylistItem, error)
	GetLastPlaylistItemByPlaylistId(context.Context, uuid.UUID) (*PlaylistItem, error)
	GetPlaylistItemCount(context.Context, uuid.UUID) (int64, error)
	GetPlaylistItemMediaById(context.Context, uuid.UUID) ([]*PlaylistItem, error)
	GetPlaylistItemMediaByIdWithFilter(
		context.Context,
		uuid.UUID,
		*PlaylistItemFilter,
	) ([]*PlaylistItem, error)
	GetUserPlaylists(context.Context, uuid.UUID, PlaylistFilter) ([]*Playlist, int64, error)
	GetPlaylistItemCounts(
		ctx context.Context,
		playlistIds []uuid.UUID,
	) (map[uuid.UUID]int64, error)

	CheckMediaExistsInPlaylists(
		ctx context.Context,
		playlistIds []uuid.UUID,
		mediaId uuid.UUID,
	) ([]*PlaylistItem, error)

	UpdatePlaylistById(context.Context, uuid.UUID, uuid.UUID, *Playlist) error
	UpdatePlaylistItem(context.Context, *PlaylistItem) error
	UpdatePlaylistItemsPosition(context.Context, []*PlaylistItem) error
	UpdatePlaylistItemPosition(context.Context, *PlaylistItem) error

	DeletePlaylistById(context.Context, uuid.UUID, uuid.UUID) error
	DeletePlaylistItemById(context.Context, uuid.UUID, uuid.UUID) error
	DeletePlaylistItemByPlaylistId(context.Context, uuid.UUID) error
	DeletePlaylistThumbnail(context.Context, uuid.UUID) error
}

type Playlist struct {
	Id           uuid.UUID          `json:"id"                      gorm:"primaryKey;type:uuid"`
	UserId       uuid.UUID          `json:"user_id"                 gorm:"type:uuid"`
	Name         string             `json:"name"`
	Metadata     JsonB              `json:"metadata,omitempty"      gorm:"type:jsonb"`
	Tags         string             `json:"tags,omitempty"`
	ThumbnailUrl string             `json:"thumbnail_url,omitempty" gorm:"-"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	ItemCount    int                `json:"item_count"              gorm:"-"`
	MediaItems   []*PlaylistItem    `json:"items"             gorm:"foreignKey:PlaylistId;references:Id"`
	Duration     float64            `json:"duration"                gorm:"-"`
	Size         int64              `json:"size"                    gorm:"-"`
	PlaylistUrl  string             `json:"playlist_url,omitempty"  gorm:"-"`
	IFrame       string             `json:"iframe,omitempty"        gorm:"-"`
	Thumbnail    *PlaylistThumbnail `json:"-"                       gorm:"foreignKey:PlaylistId;references:Id"`
	PlaylistType string             `json:"playlist_type"`
} //	@name	Playlist

type PlaylistThumbnail struct {
	PlaylistId  uuid.UUID  `json:"playlist_id"  gorm:"primaryKey;type:uuid"`
	ThumbnailId uuid.UUID  `json:"thumbnail_id" gorm:"primaryKey;type:uuid"`
	Thumbnail   *Thumbnail `json:"thumbnail"    gorm:"foreignKey:ThumbnailId;references:Id"`
}

type PlaylistItem struct {
	Id         uuid.UUID          `json:"id"`
	PlaylistId uuid.UUID          `json:"playlist_id"        gorm:"foreignKey:PlaylistId"`
	MediaId    uuid.UUID          `json:"video_id"           gorm:"foreignKey:MediaId;index:idx_playlist_item_media_id;type:uuid"`
	NextId     *uuid.UUID         `json:"next_id"            gorm:"index"`
	Next       *PlaylistItem      `json:"next,omitempty"     gorm:"foreignKey:NextId"                                             swaggerignore:"true"`
	PreviousId *uuid.UUID         `json:"previous_id"        gorm:"index"`
	Previous   *PlaylistItem      `json:"previous,omitempty" gorm:"foreignKey:PreviousId"                                         swaggerignore:"true"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
	Media      *Media             `json:"-"                  gorm:"foreignKey:MediaId"`
	MediaItem  *PlaylistItemMedia `json:"media"              gorm:"-"`
} //	@name	PlaylistItem

type PublicPlaylistObject struct {
	PlayerTheme *PlayerTheme `json:"player_theme"`
	Playlist    *Playlist    `json:"playlist"`
} //	@name	PublicPlaylistObject

type PlaylistItemMedia struct {
	ThumbnailUrl string          `json:"thumbnail_url"`
	Title        string          `json:"title"`
	Qualities    string          `json:"qualities,omitempty"`
	Size         int64           `json:"size,omitempty"`
	Duration     float64         `json:"duration,omitempty"`
	HlsUrl       string          `json:"hls_url,omitempty"`
	Chapters     []*MediaChapter `json:"chapters,omitempty"`
	Captions     []*MediaCaption `json:"captions,omitempty"`
	Description  string          `json:"description,omitempty"`
} //

type PlaylistFilter struct {
	Search       string     `json:"search"`
	SortBy       string     `json:"sort_by"`
	OrderBy      string     `json:"order_by"`
	Offset       int        `json:"offset"`
	Limit        int        `json:"limit"`
	PlaylistType string     `json:"playlist_type"`
	Metadata     []Metadata `json:"metadata"`
	Tags         []string   `json:"tags"`
}

type PlaylistItemFilter struct {
	SortBy  string `json:"sort_by"`
	OrderBy string `json:"order_by"`
	Search  string `json:"search"`
}

func NewPlaylist(
	userId uuid.UUID,
	name string,
	metadata []Metadata,
	tags []string,
	thumbnailId *uuid.UUID,
	PlaylistType string,
) (*Playlist, error) {
	newPlaylist := &Playlist{
		Id:           uuid.New(),
		UserId:       userId,
		Name:         name,
		PlaylistType: PlaylistType,
		Tags:         strings.Join(tags, ","),
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if len(metadata) != 0 {
		newPlaylist.Metadata = make(map[string]any)
		for _, meta := range metadata {
			newPlaylist.Metadata[meta.Key] = meta.Value
		}
	}

	return newPlaylist, nil
}

func NewPlaylistItem(
	playlistId, mediaId uuid.UUID,
	nextId, previousId *uuid.UUID,
) (*PlaylistItem, error) {
	return &PlaylistItem{
		Id:         uuid.New(),
		PlaylistId: playlistId,
		MediaId:    mediaId,
		NextId:     nextId,
		PreviousId: previousId,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}, nil
}

type UpdatePlaylistInput struct {
	PlaylistId uuid.UUID             `json:"playlist_id"`
	Name       *string               `json:"name"`
	Metadata   []Metadata            `json:"metadata"`
	Tags       []string              `json:"tags"`
	Thumbnail  *multipart.FileHeader `json:"thumbnail"`
}

func NewPlaylistPublicObject(playlist *Playlist, playerTheme *PlayerTheme) *PublicPlaylistObject {
	return &PublicPlaylistObject{
		Playlist:    playlist,
		PlayerTheme: playerTheme,
	}
}

func (p *Playlist) GetThumbnailUrl() string {
	var thumbnailUrl string
	if p.Thumbnail != nil {
		thumbnailUrl = fmt.Sprintf(PlayistThumbailUrlFormat, BeUrl, p.Id, "original")
	} else if len(p.MediaItems) > 0 {
		return fmt.Sprintf(
			AssetUrlFormat,
			BeUrl,
			p.MediaItems[0].MediaId,
		) + "/thumbnail?resolution=original"
	}

	return thumbnailUrl
}

func (p *Playlist) GetPlaylistUrl() string {
	return fmt.Sprintf("%s/playlist/%s", PlayerUrl, p.Id)
}

func (p *Playlist) GetIframe() string {
	playlistUrl := p.GetPlaylistUrl()
	return fmt.Sprintf(
		`<iframe src="%s" width="100%%" height="100%%" frameborder="0" scrolling="no" allowfullscreen="true"></iframe>`,
		playlistUrl,
	)
}
