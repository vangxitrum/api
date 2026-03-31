package models

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	MinWatchTime        float64 = 5
	MinPercentWatchTime         = 0.5
)

type StatisticRepository interface {
	CreateSession(ctx context.Context, session *Session) error
	CreateSessionMedia(ctx context.Context, sessionMedia *SessionMedia) error
	CreateActions(ctx context.Context, action []*Action) error
	CreateWatchInfos(ctx context.Context, watchInfos []*WatchInfo) error

	GetSessionById(ctx context.Context, sessionId uuid.UUID) (*Session, error)
	GetUncalculatedSessionMedia(ctx context.Context, cursor time.Time) ([]*SessionMedia, error)

	GetSessionMediaBySessionIdAndMediaId(
		ctx context.Context,
		sessionId, mediaId uuid.UUID,
	) (*SessionMedia, error)
	GetSessionMediaLastWatchInfo(
		ctx context.Context,
		sessionMediaId uuid.UUID,
	) (*WatchInfo, error)
	GetUserAggreagatedMetricsInAction(
		ctx context.Context,
		userId uuid.UUID,
		input GetAggregatedMetricsInput,
	) (float64, error)
	GetUserAggreagatedMetricsInWatchInfo(
		ctx context.Context,
		userId uuid.UUID,
		input GetAggregatedMetricsInput,
	) (float64, error)
	GetUserBreakdownMetricsInAction(
		ctx context.Context,
		userId uuid.UUID,
		input GetBreakdownMetricsInput,
	) ([]*MetricItem, int64, error)
	GetUserOvertimeMetricsInAction(
		ctx context.Context,
		userId uuid.UUID,
		input GetOvertimeMetricsInput,
	) ([]*MetricItem, int64, error)
	GetUserBreakdownMetricsInWatchInfo(
		ctx context.Context,
		userId uuid.UUID,
		input GetBreakdownMetricsInput,
	) ([]*MetricItem, int64, error)
	GetUserOvertimeMetricsInWatchInfo(
		ctx context.Context,
		userId uuid.UUID,
		input GetOvertimeMetricsInput,
	) ([]*MetricItem, int64, error)
	GetStatisticMedias(
		ctx context.Context,
		input GetStatisticMediasInput,
		userId uuid.UUID,
	) ([]*Media, int64, error)
	GetDataUsage(
		ctx context.Context,
		from, to time.Time,
		limit, offset uint64,
		interval string,
		userId uuid.UUID,
	) ([]*DataUsage, int64, error)

	GetUserCount(context.Context) (int64, error)
	GetNewUserCount(context.Context, TimeRange) (int64, error)
	GetTotalUserTopUps(context.Context, TimeRange) (decimal.Decimal, error)
	GetTotalUserCharge(context.Context, TimeRange) (*Billing, error)
	GetMediaCount(context.Context) (int64, error)
	GetTotalVideoWatchTime(context.Context, TimeRange) (float64, error)
	GetTotalFailQualityCount(context.Context, TimeRange) (int64, error)
	GetTotalActiveUsers(context.Context, TimeRange) (int64, error)
	GetTotalInactiveUsers(context.Context, TimeRange) (int64, error)
	GetLiveStreamCount(context.Context, TimeRange) (int64, error)

	UpdateSessionMediasStatus(ctx context.Context, sessionMediaIds []uuid.UUID, status string) error
}

type Session struct {
	Id             uuid.UUID `json:"id"              gorm:"id;primaryKey;type:uuid"`
	IP             string    `json:"ip"              gorm:"ip"`
	Continent      string    `json:"continent"       gorm:"continent"`
	Country        string    `json:"country"         gorm:"country"`
	DeviceType     string    `json:"device_type"     gorm:"device_type"`
	OperatorSystem string    `json:"operator_system" gorm:"operator_system"`
	Browser        string    `json:"browser"         gorm:"browser"`
	CreatedAt      time.Time `json:"created_at"      gorm:"created_at"`
	UserAgent      string    `json:"user_agent"      gorm:"user_agent"`
	SecUaBrowser   string    `json:"sec_ua_browser"  gorm:"sec_ua_browser"`
}

func NewSession(
	sessionId uuid.UUID,
	continent, country, ip, deviceType, operatorSystem, browser, userAgent, secUaBrowser string,
) *Session {
	return &Session{
		Id:             sessionId,
		Continent:      continent,
		Country:        country,
		IP:             ip,
		DeviceType:     deviceType,
		OperatorSystem: operatorSystem,
		Browser:        browser,
		UserAgent:      userAgent,
		SecUaBrowser:   secUaBrowser,
	}
}

type SessionMedia struct {
	Id             uuid.UUID `json:"id"              gorm:"id;primaryKey;type:uuid"`
	SessionId      uuid.UUID `json:"session_id"      gorm:"session_id;index:index_session_id;type:uuid"`
	MediaId        uuid.UUID `json:"media_id"        gorm:"media_id;index:idx_session_media_media_id;type:uuid"`
	ViewCalculated bool      `json:"view_calculated" gorm:"view_calculated"`
	Status         string    `json:"status"          gorm:"status;index:idx_status_created_at,priority:1"`
	CreatedAt      time.Time `json:"created_at"      gorm:"created_at;index:idx_status_created_at,priority:2"`
}

func NewSessionMedia(sessionId, mediaId uuid.UUID) *SessionMedia {
	return &SessionMedia{
		Id:        uuid.New(),
		SessionId: sessionId,
		MediaId:   mediaId,
		Status:    NewStatus,
	}
}

type SessionInfo struct {
	SessionId      uuid.UUID `json:"session_id"`
	IP             string    `json:"ip"`
	DeviceType     string    `json:"device_type"`
	OperatorSystem string    `json:"operator_system"`
	Browser        string    `json:"browser"`
	UserAgent      string    `json:"user_agent"`
	SecUaBrowser   string    `json:"sec_ua_browser"`
}

func GetOs(userAgent string) string {
	userAgent = strings.ToLower(userAgent)
	osKeywords := map[string][]string{
		"ios": {
			"iphone", "ipad", "ipod",
		},
		"android": {
			"android",
		},
		"windows": {
			"windows nt", "windows", "win32", "win64",
		},
		"mac osx": {
			"macintosh", "mac os x", "mac os",
		},
		"linux": {
			"linux", "x11",
		},
	}

	for os, keywords := range osKeywords {
		for _, keyword := range keywords {
			if strings.Contains(userAgent, keyword) {
				return os
			}
		}
	}

	return "unknown"
}

func GetDeviceType(userAgent string) string {
	userAgent = strings.ToLower(userAgent)
	deviceKeywords := map[string][]string{
		"phone": {
			"mobile", "android", "iphone", "ipod", "blackberry", "iemobile",
			"opera mini", "opera mobi", "phone", "palm", "windows phone",
		},
		"tablet": {
			"ipad", "tablet", "kindle", "playbook", "silk", "nexus 7", "nexus 10",
			"galaxy tab", "xoom", "sch-i800", "playbook", "tablet", "kindle",
		},
		"tv": {
			"smart-tv", "smarttv", "googletv", "appletv", "hbbtv", "pov_tv",
			"netcast", "viera", "nettv", "smarttv", "internet tv", "boxee",
			"dlnadoc", "roku", "mediabox", "dtv", "netbox",
		},
		"console": {
			"nintendo", "playstation", "xbox", "wii", "shield", "ouya",
		},
		"wearable": {
			"watch", "sm-v700", "glass", "google glass", "gear", "sm-r",
		},
		"computer": {
			"windows nt", "macintosh", "linux x86_64", "x11",
		},
	}

	for category, keywords := range deviceKeywords {
		for _, keyword := range keywords {
			if strings.Contains(userAgent, keyword) {
				return category
			}
		}
	}

	return "unknown"
}

var (
	regexDevice = regexp.MustCompile(
		`Macintosh|Windows|Linux|iPhone|iPad|iPod|Android|BlackBerry|IEMobile|Opera Mini|Mobile|Tablet`,
	)
	WHITE_LIST_COMMON = []string{
		"Chrome",
		"Chromium",
		"Edg",
		"Edge",
		"Firefox",
		"Opera",
		"OPR",
		"Presto",
	}
	WHITE_LIST_IGNORE_IOS = WHITE_LIST_COMMON
	WHITE_LIST_IOS        = append([]string{"CriOS", "FxiOS", "Safari"}, WHITE_LIST_COMMON...)
	ALL_BROWSER           = append(WHITE_LIST_IOS, WHITE_LIST_COMMON...)
	regexIOSPhone         = regexp.MustCompile(`iPhone|iPad|iPod|iOS`)
	regexAndroidPhone     = regexp.MustCompile(`Android`)
	regexPhone            = regexp.MustCompile(`Phone|Mobile`)
)

type DeviceInfo struct {
	DeviceType     string `json:"type_device"`
	OperatorSystem string `json:"operator_system"`
	Browser        string `json:"browser"`
	Original       string `json:"original"`
}

func GetDeviceInfo(userAgentString string) DeviceInfo {
	device := regexDevice.FindString(userAgentString)
	if regexIOSPhone.MatchString(userAgentString) {
		browsersDetected := findBrowsers(userAgentString, WHITE_LIST_IOS)
		browser := mapBrowser(browsersDetected)
		return DeviceInfo{
			DeviceType:     "phone",
			OperatorSystem: "ios",
			Browser:        browser,
			Original:       userAgentString,
		}
	} else if regexAndroidPhone.MatchString(userAgentString) {
		browsersDetected := findBrowsers(userAgentString, WHITE_LIST_IGNORE_IOS)
		browser := lastBrowser(browsersDetected)
		return DeviceInfo{
			DeviceType:     "phone",
			OperatorSystem: "android",
			Browser:        browser,
			Original:       userAgentString,
		}
	} else if regexPhone.MatchString(userAgentString) {
		browsersDetected := findBrowsers(userAgentString, WHITE_LIST_COMMON)
		browser := lastBrowser(browsersDetected)
		return DeviceInfo{
			DeviceType:     "phone",
			OperatorSystem: strings.ToLower(device),
			Browser:        browser,
			Original:       userAgentString,
		}
	} else {
		browsersDetected := findBrowsers(userAgentString, ALL_BROWSER)
		browser := mapBrowser(browsersDetected)
		if strings.Contains(browser, "safari") && slices.Contains(browsersDetected, "Chrome") {
			browser = "chrome"
		}
		return DeviceInfo{
			DeviceType:     "computer",
			OperatorSystem: strings.ToLower(device),
			Browser:        browser,
			Original:       userAgentString,
		}
	}
}

func findBrowsers(userAgentString string, browserList []string) []string {
	regex := regexp.MustCompile(strings.Join(browserList, "|"))
	return regex.FindAllString(userAgentString, -1)
}

func mapBrowser(browsers []string) string {
	if len(browsers) == 0 {
		return "unknown"
	}
	last := browsers[len(browsers)-1]
	switch last {
	case "CriOS":
		return "chrome(ios)"
	case "FxiOS":
		return "firefox(ios)"
	case "edg":
		return "edge"
	default:
		return strings.ToLower(last)
	}
}

func lastBrowser(browsers []string) string {
	if len(browsers) == 0 {
		return "unknown"
	}
	return strings.ToLower(browsers[len(browsers)-1])
}
