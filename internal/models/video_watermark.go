package models

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var validValueRegex = `^(\d+)(px|%)$`

type MediaWatermark struct {
	Id          uuid.UUID  `json:"id"           gorm:"primaryKey;type:uuid"`
	MediaId     uuid.UUID  `json:"media_id"     gorm:"type:uuid;references:Id"`
	Media       *Media     `json:"-"            gorm:"foreignKey:MediaId;references:Id"`
	WatermarkId uuid.UUID  `json:"watermark_id" gorm:"type:uuid;references:Id"`
	Watermark   *Watermark `json:"watermark"    gorm:"foreignKey:WatermarkId;references:Id"`
	Top         string     `json:"top"`
	Left        string     `json:"left"`
	Bottom      string     `json:"bottom"`
	Right       string     `json:"right"`
	Width       string     `json:"width"`
	Height      string     `json:"height"`
	Opacity     string     `json:"opacity"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CdnFile     *CdnFile   `json:"-"            gorm:"-"`
} //	@name	MediaWatermark

func NewMediaWatermark(
	mediaId, watermarkId uuid.UUID,
	width, height,
	top, left, bottom, right, opacity string,
) *MediaWatermark {
	return &MediaWatermark{
		Id:          uuid.New(),
		MediaId:     mediaId,
		WatermarkId: watermarkId,
		Top:         top,
		Left:        left,
		Bottom:      bottom,
		Right:       right,
		Width:       width,
		Height:      height,
		Opacity:     opacity,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

type AddWatermarkToMedia struct {
	WatermarkId uuid.UUID `json:"watermark_id"`
	MediaId     uuid.UUID `json:"media_id"`
	Top         string    `json:"top"`
	Left        string    `json:"left"`
	Bottom      string    `json:"bottom"`
	Right       string    `json:"right"`
	Width       string    `json:"width"`
	Height      string    `json:"height"`
	Opacity     string    `json:"opacity"`
}

// note: valid value format: %d% or %dpx
func (w *MediaWatermark) IsValid() bool {
	if w.Width != "" {
		if regexp.MustCompile(validValueRegex).MatchString(w.Width) == false {
			return false
		}
	}

	if w.Height != "" {
		if regexp.MustCompile(validValueRegex).MatchString(w.Height) == false {
			return false
		}
	}

	if w.Top != "" {
		if regexp.MustCompile(validValueRegex).MatchString(w.Top) == false {
			return false
		}
	}

	if w.Left != "" {
		if regexp.MustCompile(validValueRegex).MatchString(w.Left) == false {
			return false
		}
	}

	if w.Right != "" {
		if regexp.MustCompile(validValueRegex).MatchString(w.Right) == false {
			return false
		}
	}

	if w.Bottom != "" {
		if regexp.MustCompile(validValueRegex).MatchString(w.Bottom) == false {
			return false
		}
	}

	if w.Opacity != "" {
		if regexp.MustCompile(`^(\d+)%$`).MatchString(w.Opacity) == false {
			return false
		}
	}

	return true
}

func (w *MediaWatermark) GetWidth() int64 {
	return convertToValue(w.Width, w.Watermark.Width)
}

func (w *MediaWatermark) GetHeight() int64 {
	return convertToValue(w.Height, w.Watermark.Height)
}

func (w *MediaWatermark) GetTop() int64 {
	return convertToValue(w.Top, w.Watermark.Width)
}

func (w *MediaWatermark) GetLeft() int64 {
	return convertToValue(w.Left, w.Watermark.Height)
}

func (w *MediaWatermark) GetBottom() int64 {
	return convertToValue(w.Bottom, w.Watermark.Width)
}

func (w *MediaWatermark) GetRight() int64 {
	return convertToValue(w.Right, w.Watermark.Height)
}

func (w *MediaWatermark) GetOpacity() float64 {
	if w.Opacity == "" {
		return 1
	}

	var rs float64
	if strings.Contains(w.Opacity, "%") {
		pc, _ := strconv.ParseFloat(w.Opacity[:len(w.Opacity)-1], 64)
		rs = pc / 100
	}

	return rs
}

func convertToValue(input string, origin int64) int64 {
	if input == "" {
		return 0
	}

	var rs int64
	if strings.Contains(input, "px") {
		rs, _ = strconv.ParseInt(input[:len(input)-2], 10, 64)
	}

	if strings.Contains(input, "%") {
		pc, _ := strconv.ParseInt(input[:len(input)-1], 10, 64)
		rs = origin * pc / 100
	}

	return rs
}
