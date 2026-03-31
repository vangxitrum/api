package image

import (
	"fmt"
	"strings"
)

func CheckFileType(fileType string) error {
	supportedTypes := []string{"image/jpeg", "image/png"}
	for _, t := range supportedTypes {
		if strings.EqualFold(fileType, t) {
			return nil
		}
	}
	supportedExtensions := []string{"jpg", "jpeg", "png"}
	return fmt.Errorf("Only [%s] extensions are supported.", strings.Join(supportedExtensions, ", "))
}
