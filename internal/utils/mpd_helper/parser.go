package mdp_helper

import (
	"fmt"

	"github.com/nbio/xml"
)

func Unmarshal(bytes []byte) (*MPD, error) {
	var mpd MPD
	err := xml.Unmarshal(bytes, &mpd)
	if err != nil {
		return nil, fmt.Errorf("error parsing MPD: %w", err)
	}
	return &mpd, nil
}

func Marshal(mpd *MPD) ([]byte, error) {
	bytes, err := xml.MarshalIndent(mpd, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshalling MPD: %w", err)
	}

	return []byte(xml.Header + string(bytes)), nil
}
