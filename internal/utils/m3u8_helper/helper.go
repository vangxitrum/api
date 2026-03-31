package m3u8_helper

import (
	"bytes"
	"fmt"

	"github.com/grafov/m3u8"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
)

func MergeM3U8Files(media *models.Media) (*bytes.Reader, error) {
	var firstVariant *m3u8.Variant
	masterPl := m3u8.NewMasterPlaylist()
	masterPl.SetVersion(3)
	alternatives := make([]*m3u8.Alternative, 0)
	for _, q := range media.MediaQualities {
		if !(q.Status == models.DoneStatus && q.Type == models.HlsQualityType) {
			var variant *m3u8.Variant
			if q.VideoConfig != nil {
				variant = &m3u8.Variant{}
				variant.Codecs = q.VideoCodec
				variant.URI = fmt.Sprintf("%s/playlist.m3u8", q.Id)
				variant.VariantParams.Resolution = fmt.Sprintf(
					"%dx%d",
					q.VideoConfig.Width,
					q.VideoConfig.Height,
				)
				if q.AudioConfig == nil {
					variant.Codecs = fmt.Sprintf("%s,%s", q.VideoCodec, q.AudioCodec)
					variant.VariantParams.Audio = "audio"
				}

				variant.Bandwidth = uint32(q.Bandwidth)
				masterPl.Variants = append(masterPl.Variants, variant)
			} else {
				alternatives = append(alternatives, &m3u8.Alternative{
					URI:      fmt.Sprintf("%s/playlist.m3u8", q.Id),
					Type:     "AUDIO",
					GroupId:  "AUDIO",
					Name:     q.Name,
					Language: q.AudioConfig.Language,
					Default:  false,
				})
			}

			if firstVariant == nil {
				firstVariant = variant
			}
		}
	}

	if firstVariant != nil {
		firstVariant.VariantParams.Alternatives = alternatives
	}

	return bytes.NewReader(masterPl.Encode().Bytes()), nil
}
