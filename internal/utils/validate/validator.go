package validate

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/image"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
)

func IsValidUrl(urlString string) bool {
	if strings.Contains(urlString, " ") {
		return false
	}

	_, err := url.ParseRequestURI(urlString)
	if err != nil {
		return false
	}

	u, err := url.Parse(urlString)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	hostnamePattern := `^[a-zA-Z0-9.-]+$`
	hostMatch, _ := regexp.MatchString(hostnamePattern, u.Host)
	if !hostMatch {
		return false
	}

	return true
}

func IsHttpsUrl(urlString string) bool {
	u, err := url.ParseRequestURI(urlString)
	if err != nil {
		return false
	}

	if u.Scheme != "https" || u.Host == "" {
		return false
	}

	return true
}

func CheckAllowedPath(path string, method string) bool {
	patterns := []struct {
		pathRegex *regexp.Regexp
		method    string
	}{
		{regexp.MustCompile(`^/api/media/create$`), http.MethodPost},
		{regexp.MustCompile(`media/.+/part`), http.MethodPost},
		{regexp.MustCompile(`media/.+/complete`), http.MethodGet},
	}

	for _, pattern := range patterns {
		if pattern.pathRegex.MatchString(path) && method == pattern.method {
			return true
		}
	}

	return false
}

func IsValidHexOrRGBAColor(color string) bool {
	if match, _ := regexp.MatchString(`^#[0-9A-Fa-f]{6}([0-9A-Fa-f]{2})?$`, color); match {
		return true
	}

	if strings.HasPrefix(color, "rgba(") && strings.HasSuffix(color, ")") {
		values := strings.Split(strings.TrimSuffix(strings.TrimPrefix(color, "rgba("), ")"), ",")
		if len(values) != 4 {
			return false
		}

		for i, v := range values {
			v = strings.TrimSpace(v)
			if i < 3 {
				num, err := strconv.Atoi(v)
				if err != nil || num < 0 || num > 255 {
					return false
				}
			} else {
				alpha, err := strconv.ParseFloat(v, 64)
				if err != nil || alpha < 0 || alpha > 1 {
					return false
				}
			}
		}
		return true
	}

	return false
}

func IsValidPixelValue(value string) bool {
	if match, _ := regexp.MatchString(`^\d+px$`, value); match {
		return true
	}

	return false
}

func AreThemePixelsValid(theme models.Theme) bool {
	values := []struct {
		name  string
		value string
	}{
		{"ControlBarHeight", theme.ControlBarHeight},
		{"ProgressBarHeight", theme.ProgressBarHeight},
		{"ProgressBarCircleSize", theme.ProgressBarCircleSize},
	}

	for _, field := range values {
		if field.value != "" {
			if !IsValidPixelValue(field.value) {
				return false
			}
		}
	}

	return true
}

func AreThemeColorsValid(theme models.Theme) bool {
	colorFields := []struct {
		name  string
		value string
	}{
		{"MainColor", theme.MainColor},
		{"TextColor", theme.TextColor},
		{"TextTrackColor", theme.TextTrackColor},
		{"TextTrackBackground", theme.TextTrackBackground},
		{"ControlBarBackgroundColor", theme.ControlBarBackgroundColor},
		{"MenuBackGroundColor", theme.MenuBackGroundColor},
		{"MenuItemBackGroundHover", theme.MenuItemBackGroundHover},
	}

	for _, field := range colorFields {
		if field.value != "" {
			if !IsValidHexOrRGBAColor(field.value) {
				return false
			}
		}
	}

	return true
}

func IsValidAdminEmail(email string) bool {
	return models.ValidEmail[email]
}

func IsValidateThumbnail(file *multipart.FileHeader) error {
	src, err := file.Open()
	if err != nil {
		return response.NewInternalServerError(err)
	}
	defer src.Close()

	buff := make([]byte, 512)
	if _, err = src.Read(buff); err != nil {
		return response.NewInternalServerError(err)
	}
	fileType := http.DetectContentType(buff)

	if err := image.CheckFileType(fileType); err != nil {
		return response.NewHttpError(http.StatusBadRequest, err)
	}
	return nil
}

func IsValidateTags(tags *[]string) error {
	if len(*tags) > models.TagMaxItems {
		return response.NewHttpError(
			http.StatusBadRequest,
			nil,
			fmt.Sprintf("Number of tags must be less than %d.", models.TagMaxItems),
		)
	}

	filteredTags := make([]string, 0, len(*tags))
	mapTags := make(map[string]bool)
	for _, tag := range *tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if len(tag) > models.TagMaxLen {
			return response.NewHttpError(
				http.StatusBadRequest,
				nil,
				fmt.Sprintf("Tag length must be less than %d characters.", models.TagMaxLen),
			)
		}

		if _, ok := mapTags[tag]; !ok {
			filteredTags = append(filteredTags, tag)
			mapTags[tag] = true
		}
	}

	slices.Sort(filteredTags)
	*tags = filteredTags
	return nil
}

func IsValidateMetadata(metadata *[]models.Metadata) error {
	if len(*metadata) > models.MetadataMaxItems {
		return response.NewHttpError(
			http.StatusBadRequest,
			nil,
			fmt.Sprintf("Number of metadata must be less than %d.", models.MetadataMaxItems),
		)
	}

	filteredMetadata := make([]models.Metadata, 0, len(*metadata))
	mapMetadata := make(map[string]bool)
	for _, meta := range *metadata {
		meta.Key = strings.TrimSpace(strings.ToLower(meta.Key))
		meta.Value = strings.TrimSpace(strings.ToLower(meta.Value))
		if len(meta.Key) > models.MetadataMaxLen || len(meta.Value) > models.MetadataMaxLen {
			return response.NewHttpError(
				http.StatusBadRequest,
				nil,
				fmt.Sprintf(
					"Metadata key and value length must be less than %d characters.",
					models.MetadataMaxLen,
				),
			)
		}

		if _, ok := mapMetadata[meta.Key]; !ok {
			filteredMetadata = append(filteredMetadata, meta)
			mapMetadata[meta.Key] = true
		}
	}

	*metadata = filteredMetadata
	return nil
}

func NormalizeRTMP(url string) string {
	liveIndex := strings.Index(url, "live/")
	if liveIndex != -1 {
		afterLiveIndex := liveIndex + len("live/")
		for afterLiveIndex < len(url) && url[afterLiveIndex] == '/' {
			afterLiveIndex++
		}
		url = url[:liveIndex+len("live/")] + url[afterLiveIndex:]
	}
	return url
}

func IsValidProtocolsSupported(urlStream string) bool {
	parsedURL, err := url.Parse(urlStream)

	if err != nil ||
		parsedURL.Scheme != "rtmp" {
		return false
	}

	// Prevent command injection to ffmpeg command. This is a first check, mediamtx will check again.
	return !strings.Contains(urlStream, " ") && strings.HasPrefix(parsedURL.String(), "rtmp://")
}
