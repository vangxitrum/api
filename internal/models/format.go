package models

import (
	"context"

	"github.com/google/uuid"
)

type FormatRepository interface {
	Create(context.Context, *MediaFormat) error

	GetFormatByMediaId(context.Context, uuid.UUID) (*MediaFormat, error)

	DeleteFormatByMediaId(context.Context, uuid.UUID) error
}

type MediaFormat struct {
	Id             uuid.UUID `json:"format_id"        gorm:"primaryKey;id;type:uuid"`
	MediaId        uuid.UUID `json:"media_id"         gorm:"type:uuid;references:Id;index:idx_format_media_id"`
	Media          *Media    `json:"media"            gorm:"foreignKey:MediaId"`
	Filename       string    `json:"filename"`
	NbStreams      int       `json:"nb_streams"`
	NbPrograms     int       `json:"nb_programs"`
	FormatName     string    `json:"format_name"`
	FormatLongName string    `json:"format_long_name"`
	StartTime      string    `json:"start_time"`
	Duration       string    `json:"duration"`
	Size           string    `json:"size"`
	BitRate        string    `json:"bit_rate"`
	ProbeScore     int       `json:"probe_score"`
}
