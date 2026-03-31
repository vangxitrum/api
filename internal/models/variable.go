package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ImageSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

var (
	InputPath  = "./input"
	OutputPath = "./output"

	StreamPublicKeyHeader = "stream-public-key"
	StreamSecretKeyHeader = "stream-secret-key"
	StreamTraceIdHeader   = "stream-trace-id"
	AdminApiKeyHeader     = "Admin-Api-Key"

	DemoVideoId uuid.UUID

	DepositType uint8 = 0
	BillingType uint8 = 1

	CdnBlockSize = 100 * 1000 * 1000 // 10MB

	NotEnoughBalanceMessage = "Not enough balance"

	BeUrl                    = "http://localhost:8080"
	FeUrl                    = "http://10.0.0.181:3031"
	PlayerUrl                = "http://10.0.0.181:3333"
	SegmentUrlFormat         = "%s/api/media/vod/%s/%s"
	PlaylistUrlFormat        = "%s/api/media/vod/%s/playlist.m3u8?range=%d,%d&type=%s"
	AssetUrlFormat           = "%s/api/media/%s"
	PlayerLogoUlrFormat      = "%s/api/players/%s"
	PlayistThumbailUrlFormat = "%s/api/playlists/%s/thumbnail?resolution=%s"

	BetterStackToken = ""

	MaxMediaPendingTime    = 24 * time.Hour
	MaxNotFoundPendingTime = 15 * 24 * time.Hour

	NewStatus          = "new"
	WaitingStatus      = "waiting"
	PrepaidStatus      = "prepaid"
	ActiveStatus       = "active"
	UploadedStatus     = "uploaded"
	TranscodingStatus  = "transcoding"
	UploadingStatus    = "uploading"
	DeletingStatus     = "deleting"
	DeletedStatus      = "deleted"
	HiddenStatus       = "hidden"
	DoneStatus         = "done"
	FailStatus         = "fail"
	PendingStatus      = "pending"
	SuccessStatus      = "success"
	BlockedStatus      = "blocked"
	CountedStatus      = "counted"
	UnCountedStatus    = "uncounted"
	TranscribingStatus = "transcribing"
	LowBalanceStatus   = "low_balance"
	OutOfBalanceStatus = "out_of_balance"
	ProcessingStatus   = "processing"
	ExpiredStatus      = "expired"

	H265VideoMpdCodec = "hvc1.1.6.L63.90"
	VideoMediaType    = "video"
	AudioMediaType    = "audio"
	StreamMediaType   = "live_stream"
	CaptionFormat     = "vtt"
	AudioFormat       = "mp3"

	DefaultCaptionLanguage = map[string]string{
		"Hindi":   "hi",
		"Chinese": "zh",
	}

	DefaultThumbnailSize = []ImageSize{
		{Width: 480, Height: 270},
		{Width: 768, Height: 432},
		{Width: 1024, Height: 576},
		{Width: 1280, Height: 720},
	}
	DefaultMediaQualities = []string{"360p", "720p", "1080p"}

	DefaultAudioConfig = []*QualityConfig{
		DefaultConfigMapping["standard"],
		DefaultConfigMapping["good"],
		DefaultConfigMapping["highest"],
	}

	DefaultVideoConfig = []*QualityConfig{
		DefaultConfigMapping["360p"],
		DefaultConfigMapping["720p"],
		DefaultConfigMapping["1080p"],
	}

	DefaultConfigMapping = map[string]*QualityConfig{
		"standard": {
			Resolution:    "standard",
			Type:          HlsQualityType,
			ContainerType: MpegtsContainerType,
			AudioConfig: &AudioConfig{
				Codec:      AacCodec,
				Bitrate:    128_000,
				SampleRate: 44100,
				Channels:   "2",
			},
		},
		"good": {
			Resolution:    "good",
			Type:          HlsQualityType,
			ContainerType: MpegtsContainerType,
			AudioConfig: &AudioConfig{
				Codec:      AacCodec,
				Bitrate:    192_000,
				SampleRate: 44100,
				Channels:   "2",
			},
		},
		"highest": {
			Resolution:    "highest",
			Type:          HlsQualityType,
			ContainerType: MpegtsContainerType,
			AudioConfig: &AudioConfig{
				Codec:      AacCodec,
				Bitrate:    320_000,
				SampleRate: 44100,
				Channels:   "2",
			},
		},
		"lossless": {
			Resolution:    "lossless",
			Type:          HlsQualityType,
			ContainerType: MpegtsContainerType,
			AudioConfig: &AudioConfig{
				Codec:      AacCodec,
				Bitrate:    1_411_000,
				SampleRate: 44100,
				Channels:   "2",
			},
		},
		"240p": {
			Resolution:    "240p",
			Type:          HlsQualityType,
			ContainerType: MpegtsContainerType,
			VideoConfig: &VideoConfig{
				Codec:   H264Codec,
				Bitrate: 500_000,
				Width:   426,
				Height:  240,
			},
			AudioConfig: &AudioConfig{
				Codec:      AacCodec,
				Bitrate:    128_000,
				SampleRate: 44100,
			},
		},
		"360p": {
			Resolution:    "360p",
			Type:          HlsQualityType,
			ContainerType: MpegtsContainerType,
			VideoConfig: &VideoConfig{
				Codec:   H264Codec,
				Bitrate: 750_000,
				Width:   640,
				Height:  360,
			},
			AudioConfig: &AudioConfig{
				Codec:      AacCodec,
				Bitrate:    128_000,
				SampleRate: 44100,
			},
		},
		"480p": {
			Resolution:    "480p",
			Type:          HlsQualityType,
			ContainerType: "mpegts",
			VideoConfig: &VideoConfig{
				Codec:   H264Codec,
				Bitrate: 1_200_000,
				Width:   854,
				Height:  480,
			},
			AudioConfig: &AudioConfig{
				Codec:      AacCodec,
				Bitrate:    128_000,
				SampleRate: 44100,
			},
		},
		"720p": {
			Resolution:    "720p",
			Type:          HlsQualityType,
			ContainerType: MpegtsContainerType,
			VideoConfig: &VideoConfig{
				Codec:   H264Codec,
				Bitrate: 3_000_000,
				Width:   1280,
				Height:  720,
			},
			AudioConfig: &AudioConfig{
				Codec:      AacCodec,
				Bitrate:    128_000,
				SampleRate: 44100,
			},
		},
		"1080p": {
			Resolution:    "1080p",
			Type:          HlsQualityType,
			ContainerType: MpegtsContainerType,
			VideoConfig: &VideoConfig{
				Codec:   H264Codec,
				Bitrate: 5_000_000,
				Width:   1920,
				Height:  1080,
			},
			AudioConfig: &AudioConfig{
				Codec:      AacCodec,
				Bitrate:    128_000,
				SampleRate: 44100,
			},
		},
		"1440p": {
			Resolution:    "1440p",
			Type:          HlsQualityType,
			ContainerType: MpegtsContainerType,
			VideoConfig: &VideoConfig{
				Codec:   H264Codec,
				Bitrate: 8_000_000,
				Width:   2560,
				Height:  1440,
			},
			AudioConfig: &AudioConfig{
				Codec:      AacCodec,
				Bitrate:    128_000,
				SampleRate: 44100,
			},
		},
		"2160p": {
			Resolution:    "2160p",
			Type:          HlsQualityType,
			ContainerType: MpegtsContainerType,
			VideoConfig: &VideoConfig{
				Codec:   H264Codec,
				Bitrate: 15_000_000,
				Width:   3840,
				Height:  2160,
			},
			AudioConfig: &AudioConfig{
				Codec:      AacCodec,
				Bitrate:    128_000,
				SampleRate: 44100,
			},
		},
		"4320p": {
			Resolution:    "4320p",
			Type:          HlsQualityType,
			ContainerType: MpegtsContainerType,
			VideoConfig: &VideoConfig{
				Codec:   H264Codec,
				Bitrate: 20_000_000,
				Width:   7680,
				Height:  4320,
			},
			AudioConfig: &AudioConfig{
				Codec:      AacCodec,
				Bitrate:    128_000,
				SampleRate: 44100,
			},
		},
	}

	MaxH265Resolution int32 = 4320
	MaxH264Resolution int32 = 2160

	DefaultSegmentDuration int32 = 6
	MaxSegmentDuration     int32 = 10

	DefaultSegmentType = HlsQualityType

	H264Codec = "h264"
	H265Codec = "h265"
	AacCodec  = "aac"

	MpegtsContainerType = "mpegts"
	Mp4ContainerType    = "mp4"
	Fmp4ContainerType   = "fmp4"
	WebmContainerType   = "webm"

	MomoChannel   = "1"
	StereoChannel = "2"
	Channel21     = "3"
	QuadChannel   = "4"
	Channel50     = "5"
	Channel51     = "6"
	Channel61     = "7"
	Channel71     = "8"

	ValidQualityTypes = map[string]bool{
		HlsQualityType:  true,
		DashQualityType: true,
	}

	ValidAudioChannels = map[string]bool{
		MomoChannel:   true,
		StereoChannel: true,
		Channel21:     true,
		QuadChannel:   true,
		Channel50:     true,
		Channel51:     true,
		Channel61:     true,
		Channel71:     true,
	}

	ValidVideoCodecs = map[string]bool{
		H264Codec: true,
		H265Codec: true,
	}

	ValidAudioCodecs = map[string]bool{
		AacCodec: true,
	}

	ValidContainerType = map[string]map[string]bool{
		HlsQualityType: {
			MpegtsContainerType: true,
			Fmp4ContainerType:   true,
		},
		DashQualityType: {
			Mp4ContainerType:  true,
			WebmContainerType: true,
		},
	}

	ValidLiveStreamStatus = map[string]bool{
		NewStatus:         true,
		TranscodingStatus: true,
		DeletedStatus:     true,
		DoneStatus:        true,
		FailStatus:        true,
	}

	ValidSampleRates = map[int32]bool{
		8000:  true,
		11025: true,
		16000: true,
		22050: true,
		32000: true,
		44100: true,
		48000: true,
		88200: true,
		96000: true,
	}

	ValidMediaQualities = map[string]bool{
		"240p":  true,
		"360p":  true,
		"480p":  true,
		"720p":  true,
		"1080p": true,
		"1440p": true,
		"2160p": true,
		"4320p": true,
	}

	MaxVideoQualityBitrates = map[string]int64{
		"240p":  700_000,
		"360p":  1_200_000,
		"480p":  2_000_000,
		"720p":  4_000_000,
		"1080p": 6_000_000,
		"1440p": 12_000_000,
		"2160p": 30_000_000,
		"4320p": 60_000_000,
	}

	MaxAudioBitrate int64 = 256_000

	MediaQualitiesTranscodePriceMapping = map[string]float64{
		AudioType: 0.01,
		"144p":    0.01,
		"240p":    0.01,
		"360p":    0.01,
		"480p":    0.01,
		"720p":    0.01,
		"1080p":   0.01,
		"1440p":   0.02,
		"2160p":   0.05,
		"4320p":   0.06,
	}

	SupportedVideoMimetypeMapping = map[string]bool{
		"video/mp4":        true,
		"video/x-matroska": true,
		"video/x-msvideo":  true,
		"video/quicktime":  true,
		"video/x-flv":      true,
		"video/webm":       true,
		"video/x-m4v":      true,
		"video/x-ms-wmv":   true,
		"video/x-ms-asf":   true,
		"video/x-f4v":      true,
		"video/ogg":        true,
	}

	SupportedAudioMimetypeMapping = map[string]bool{
		"video/mp4":        true,
		"video/x-matroska": true,
		"video/x-msvideo":  true,
		"video/quicktime":  true,
		"video/x-flv":      true,
		"video/webm":       true,
		"video/x-m4v":      true,
		"video/x-ms-wmv":   true,
		"video/x-ms-asf":   true,
		"video/x-f4v":      true,
		"video/ogg":        true,

		"audio/aac":  true,
		"audio/mp4":  true,
		"audio/mpeg": true,
		"audio/opus": true,
		"audio/webm": true,
		"audio/ogg":  true,
		"audio/wav":  true,
		"audio/eac3": true,
		"audio/mp2t": true,
		"audio/aiff": true,
		"audio/alac": true,
	}

	ValidImageTypes = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
	}

	TagMaxLen         = 255
	TagMaxItems       = 50
	MetadataMaxLen    = 255
	MetadataMaxItems  = 50
	DescriptionMaxLen = 1000
	TitleMaxLen       = 255

	MaxMediaSize int64 = 2 * 1000 * 1000 * 1000 * 1000 // 2TB

	MaxPartSize      int64 = 200 * 1000 * 1000 // 200MB
	MaxThumbnailSize int64 = 8 * 1000 * 1000   // 8MB
	MaxVTTFileSize   int64 = 50 * 1000 * 1000  // 50MB
	MaxAudioFileSize int64 = 200 * 1000 * 1000

	PageSizeLimit int = 100

	MaxRetriesSignUp      int = 10
	MaxRetriesLogin       int = 10
	MaxRetriesDownLoadMp3 int = 3

	ValidMediaStatus = []string{
		NewStatus,
		DoneStatus,
		TranscodingStatus,
		FailStatus,
		DeletedStatus,
	}

	MaxTtl int64 = 2147483647
	MinTtl int64 = 0

	DefaultOrderBy          = "asc"
	DefaultMediaSortBy      = "created_at"
	ValidMediaSortByColumns = map[string]bool{
		"created_at": true,
		"updated_at": true,
		"title":      true,
		"size":       true,
		"status":     true,
		"view":       true,
	}

	DefaultLanguage      = "English"
	DefaultLanguage3Word = "eng"
	DefaultLanguage2Word = "en"
	LanguageMapping      = map[string]string{
		"ara": "ar", // Arabic
		"bul": "bg", // Bulgarian
		"cat": "ca", // Catalan
		"ces": "cs", // Czech
		"dan": "da", // Danish
		"deu": "de", // German
		"ell": "el", // Greek
		"eng": "en", // English
		"spa": "es", // Spanish
		"est": "et", // Estonian
		"fin": "fi", // Finnish
		"fra": "fr", // French
		"gle": "ga", // Irish
		"hrv": "hr", // Croatian
		"hun": "hu", // Hungarian
		"ind": "id", // Indonesian
		"ita": "it", // Italian
		"jpn": "ja", // Japanese
		"kor": "ko", // Korean
		"lav": "lv", // Latvian
		"lit": "lt", // Lithuanian
		"nld": "nl", // Dutch
		"nor": "no", // Norwegian
		"pol": "pl", // Polish
		"por": "pt", // Portuguese
		"ron": "ro", // Romanian
		"rus": "ru", // Russian
		"slk": "sk", // Slovak
		"slv": "sl", // Slovenian
		"srp": "sr", // Serbian
		"swe": "sv", // Swedish
		"tha": "th", // Thai
		"tur": "tr", // Turkish
		"ukr": "uk", // Ukrainian
		"vie": "vi", // Vietnamese
		"zho": "zh", // Chinese
	}

	NameLength              = 255
	MaxPageLimit     uint64 = 100
	MaxLogoSize      int64  = 102400
	MaxLogoWidth     int    = 200
	MaxLogoHeight    int    = 100
	DefaultPageLimit int    = 25

	LanToLanguageMapping = map[string]string{
		"af":  "Afrikaans",
		"sq":  "Albanian",
		"gsw": "Alsatian",
		"am":  "Amharic",
		"ar":  "Arabic",
		"hy":  "Armenian",
		"as":  "Assamese",
		"az":  "Azerbaijani",
		"bn":  "Bangla",
		"ba":  "Bashkir",
		"eu":  "Basque",
		"be":  "Belarusian",
		"bs":  "Bosnian",
		"br":  "Breton",
		"bg":  "Bulgarian",
		"my":  "Burmese",
		"ca":  "Catalan",
		"ceb": "Cebuano",
		"ku":  "Central Kurdish",
		"ccp": "Chakma",
		"chr": "Cherokee",
		"zh":  "Chinese",
		"co":  "Corsican",
		"hr":  "Croatian",
		"cs":  "Czech",
		"da":  "Danish",
		"prs": "Dari",
		"dv":  "Divehi",
		"nl":  "Dutch",
		"dz":  "Dzongkha",
		"en":  "English",
		"et":  "Estonian",
		"fo":  "Faroese",
		"fil": "Filipino",
		"fi":  "Finnish",
		"fr":  "French",
		"fy":  "Frisian",
		"ff":  "Fulah",
		"gl":  "Galician",
		"ka":  "Georgian",
		"de":  "German",
		"el":  "Greek",
		"gu":  "Gujarati",
		"ha":  "Hausa",
		"haw": "Hawaiian",
		"he":  "Hebrew",
		"hi":  "Hindi",
		"hu":  "Hungarian",
		"is":  "Icelandic",
		"ig":  "Igbo",
		"smn": "Inari Sami",
		"id":  "Indonesian",
		"iu":  "Inuktitut",
		"ga":  "Irish",
		"xh":  "isiXhosa",
		"zu":  "isiZulu",
		"it":  "Italian",
		"ja":  "Japanese",
		"quc": "K'iche'",
		"kl":  "Kalaallisut",
		"kn":  "Kannada",
		"kk":  "Kazakh",
		"km":  "Khmer",
		"rw":  "Kinyarwanda",
		"sw":  "Kiswahili",
		"kok": "Konkani",
		"ko":  "Korean",
		"ky":  "Kyrgyz",
		"lo":  "Lao",
		"lv":  "Latvian",
		"lt":  "Lithuanian",
		"dsb": "Lower Sorbian",
		"smj": "Lule Sami",
		"lb":  "Luxembourgish",
		"mk":  "Macedonian",
		"ms":  "Malay",
		"ml":  "Malayalam",
		"mt":  "Maltese",
		"mi":  "Maori",
		"arn": "Mapuche",
		"mr":  "Marathi",
		"moh": "Mohawk",
		"mn":  "Mongolian",
		"ne":  "Nepali",
		"se":  "Northern Sami",
		"nb":  "Norwegian (Bokmål)",
		"nn":  "Norwegian (Nynorsk)",
		"oc":  "Occitan",
		"or":  "Odia",
		"ps":  "Pashto",
		"fa":  "Persian",
		"pl":  "Polish",
		"pt":  "Portuguese",
		"pa":  "Punjabi",
		"quz": "Quechua",
		"ro":  "Romanian",
		"rm":  "Romansh",
		"ru":  "Russian",
		"sah": "Sakha",
		"sa":  "Sanskrit",
		"gd":  "Scottish Gaelic",
		"sr":  "Serbian",
		"nso": "Sesotho sa Leboa",
		"tn":  "Setswana",
		"sd":  "Sindhi",
		"si":  "Sinhala",
		"sms": "Skolt Sami",
		"sk":  "Slovak",
		"sl":  "Slovenian",
		"sma": "Southern Sami",
		"es":  "Spanish",
		"zgh": "Standard Moroccan Tamazight",
		"sv":  "Swedish",
		"syr": "Syriac",
		"tg":  "Tajik",
		"ta":  "Tamil",
		"tt":  "Tatar",
		"te":  "Telugu",
		"th":  "Thai",
		"bo":  "Tibetan",
		"ti":  "Tigrinya",
		"tr":  "Turkish",
		"tk":  "Turkmen",
		"uk":  "Ukrainian",
		"hsb": "Upper Sorbian",
		"ur":  "Urdu",
		"ug":  "Uyghur",
		"uz":  "Uzbek",
		"vi":  "Vietnamese",
		"cy":  "Welsh",
		"wo":  "Wolof",
		"ii":  "Yi",
		"yo":  "Yoruba",
	}

	ValidLanguageMapping = map[string]bool{
		"af":  true,
		"sq":  true,
		"gsw": true,
		"am":  true,
		"ar":  true,
		"hy":  true,
		"as":  true,
		"az":  true,
		"bn":  true,
		"ba":  true,
		"eu":  true,
		"be":  true,
		"bs":  true,
		"br":  true,
		"bg":  true,
		"my":  true,
		"ca":  true,
		"ceb": true,
		"ku":  true,
		"ccp": true,
		"chr": true,
		"zh":  true,
		"co":  true,
		"hr":  true,
		"cs":  true,
		"da":  true,
		"prs": true,
		"dv":  true,
		"nl":  true,
		"dz":  true,
		"en":  true,
		"et":  true,
		"fo":  true,
		"fil": true,
		"fi":  true,
		"fr":  true,
		"fy":  true,
		"ff":  true,
		"gl":  true,
		"ka":  true,
		"de":  true,
		"el":  true,
		"gu":  true,
		"ha":  true,
		"haw": true,
		"he":  true,
		"hi":  true,
		"hu":  true,
		"is":  true,
		"ig":  true,
		"smn": true,
		"id":  true,
		"iu":  true,
		"ga":  true,
		"xh":  true,
		"zu":  true,
		"it":  true,
		"ja":  true,
		"quc": true,
		"kl":  true,
		"kn":  true,
		"kk":  true,
		"km":  true,
		"rw":  true,
		"sw":  true,
		"kok": true,
		"ko":  true,
		"ky":  true,
		"lo":  true,
		"lv":  true,
		"lt":  true,
		"dsb": true,
		"smj": true,
		"lb":  true,
		"mk":  true,
		"ms":  true,
		"ml":  true,
		"mt":  true,
		"mi":  true,
		"arn": true,
		"mr":  true,
		"moh": true,
		"mn":  true,
		"ne":  true,
		"se":  true,
		"nb":  true,
		"nn":  true,
		"oc":  true,
		"or":  true,
		"ps":  true,
		"fa":  true,
		"pl":  true,
		"pt":  true,
		"pa":  true,
		"quz": true,
		"ro":  true,
		"rm":  true,
		"ru":  true,
		"sah": true,
		"sa":  true,
		"gd":  true,
		"sr":  true,
		"nso": true,
		"tn":  true,
		"sd":  true,
		"si":  true,
		"sms": true,
		"sk":  true,
		"sl":  true,
		"sma": true,
		"es":  true,
		"zgh": true,
		"sv":  true,
		"syr": true,
		"tg":  true,
		"ta":  true,
		"tt":  true,
		"te":  true,
		"th":  true,
		"bo":  true,
		"ti":  true,
		"tr":  true,
		"tk":  true,
		"uk":  true,
		"hsb": true,
		"ur":  true,
		"ug":  true,
		"uz":  true,
		"vi":  true,
		"cy":  true,
		"wo":  true,
		"ii":  true,
		"yo":  true,
	}

	PlayMetric           = "play"
	PlayRateMetric       = "play_rate"
	PlayTotalMetric      = "play_total"
	StartMetric          = "start"
	EndMetric            = "end"
	ImpressionMetric     = "impression"
	ImpressionTimeMetric = "impression_time"
	WatchTimeMetric      = "watch_time"
	ViewMetric           = "view"
	RetentionMetric      = "retention"

	MediaIdBreakdown        = "media-id"
	MediaTypeBreakdown      = "media-type"
	ContinentBreakdown      = "continent"
	CountryBreakdown        = "country"
	DeviceTypeBreakdown     = "device-type"
	OperatorSystemBreakdown = "operator-system"
	BrowserBreakdown        = "browser"

	BreakdownMapping = map[string]string{
		MediaIdBreakdown:        "media_id",
		MediaTypeBreakdown:      "media_type",
		ContinentBreakdown:      "continent",
		CountryBreakdown:        "country",
		DeviceTypeBreakdown:     "device_type",
		OperatorSystemBreakdown: "operator_system",
		BrowserBreakdown:        "browser",
	}

	ValidAggregatedMetrics = map[string]bool{
		PlayMetric:           true,
		StartMetric:          true,
		EndMetric:            true,
		ImpressionMetric:     true,
		ImpressionTimeMetric: true,
		WatchTimeMetric:      true,
		ViewMetric:           true,
	}

	ValidBreakdownMetrics = map[string]bool{
		PlayMetric:       true,
		PlayRateMetric:   true,
		PlayTotalMetric:  true,
		StartMetric:      true,
		EndMetric:        true,
		ImpressionMetric: true,
		WatchTimeMetric:  true,
		ViewMetric:       true,
		RetentionMetric:  true,
	}

	ValidBreakdowns = map[string]bool{
		MediaIdBreakdown:        true,
		MediaTypeBreakdown:      true,
		ContinentBreakdown:      true,
		CountryBreakdown:        true,
		DeviceTypeBreakdown:     true,
		OperatorSystemBreakdown: true,
		BrowserBreakdown:        true,
	}

	ValidBreakdownSortBy = map[string]bool{
		"metric_value":    true,
		"dimension_value": true,
	}

	ValidOvertimeSortBy = map[string]bool{
		"metric_value": true,
		"emitted_at":   true,
	}

	ValidContinents = map[string]bool{
		"AS": true, // Asia
		"EU": true, // Europe
		"AF": true, // Africa
		"NA": true, // North America
		"SA": true, // South America
		"AN": true, // Antarctica
		"AZ": true, // Australia
	}

	ValidDeviceTypes = map[string]bool{
		"computer": true,
		"phone":    true,
		"tablet":   true,
		"tv":       true,
		"console":  true,
		"wearable": true,
		"unknown":  true,
	}

	ValidOperatorSystems = map[string]bool{
		"windows": true,
		"mac osx": true,
		"linux":   true,
		"ios":     true,
		"android": true,
	}

	ValidBrowsers = map[string]bool{
		"chrome":  true,
		"firefox": true,
		"opera":   true,
		"edge":    true,
		"safari":  true,
	}

	ValidIntervals = map[string]bool{
		"day":   true,
		"hour":  true,
		"week":  true,
		"month": true,
	}

	CountAggregation   = "count"
	RateAggregation    = "rate"
	TotalAggregation   = "total"
	AvarageAggregation = "average"
	SumAggregation     = "sum"
	ViewAggregation    = "view"

	ValidAggregations = map[string]bool{
		CountAggregation:   true,
		RateAggregation:    true,
		TotalAggregation:   true,
		AvarageAggregation: true,
		SumAggregation:     true,
		ViewAggregation:    true,
	}

	MetrixToAggregationMapping = map[string][]string{
		PlayMetric: {
			CountAggregation,
			RateAggregation,
			TotalAggregation,
		},
		StartMetric:          {CountAggregation},
		EndMetric:            {CountAggregation},
		ImpressionMetric:     {CountAggregation},
		ImpressionTimeMetric: {AvarageAggregation, SumAggregation},
		WatchTimeMetric:      {AvarageAggregation, SumAggregation},
		ViewMetric:           {CountAggregation},
	}

	OrderMap = map[string]bool{
		"asc":  true,
		"desc": true,
	}

	SortByMap = map[string]bool{
		"created_at": true,
		"name":       true,
	}

	WebhookSortByMap = map[string]bool{
		"created_at": true,
		"name":       true,
		"url":        true,
	}

	PlaylistSortByMap = map[string]bool{
		"created_at": true,
		"title":      true,
		"status":     true,
	}

	PlaylistItemSortByMap = map[string]bool{
		"created_at": true,
		"title":      true,
		"duration":   true,
	}

	UploadRateLimit    = 100
	WritesRateLimit    = 200
	ReadsRateLimit     = 500
	RetryAfterDuration = time.Minute

	ExpiredEmailTime                      = 15 * time.Minute
	ExpiredEmailLowBalanceTime            = 7 * 24 * time.Hour
	ExpiredEmailOutOfBalanceTime          = 7 * 24 * time.Hour
	HubCostPerStorage            int64    = 21361
	HubCostPerDelivery           int64    = 4750000
	BytesPerTB                            = 1e12
	AdminMailList                []string = []string{}

	MaxPlaylistItemCount int64 = 100

	DeliveryCase = "delivery"
	StorageCase  = "storage"

	Second = "second"
	Minute = "minute"
	Hour   = "hour"

	MaxDescriptionLength = 1000

	ValidContentReportReason = map[ContentReportReason]bool{
		ReasonSexualContent:        true,
		ReasonViolence:             true,
		ReasonHatefulOrAbusive:     true,
		ReasonHarassmentOrBullying: true,
		ReasonHarmfulOrDangerous:   true,
		ReasonMisinformation:       true,
		ReasonChildAbuse:           true,
		ReasonPromotesTerrorism:    true,
		ReasonSpamOrMisleading:     true,
		ReasonCopyright:            true,
		ReasonCaptionsIssue:        true,
	}

	ValidMediaTypes = map[string]bool{
		VideoMediaType:  true,
		StreamMediaType: true,
	}

	ValidContentReportStatus = map[string]bool{
		"new":      true,
		"resolved": true,
	}

	ValidEmail = map[string]bool{
		"tuantq666@gmail.com":  true,
		"dat.nguyen@aioz.io":   true,
		"bhvinh3004@gmail.com": true,
		"tue.phan@aioz.io":     true,
		"dclonec001@gmail.com": true,
	}

	MaxLiveStreamingDuration = 12 * time.Hour

	TestAccountCode = "111111"
	TestAccounts    = map[string]bool{
		"pentestine@w3stream.xyz": true,
	}

	BilledCdnTypes = []string{
		CdnM3u8FileType,
		CdnSourceFileType,
		CdnVideoContentType,
		CdnVideoPlaylistType,
		CdnAudioContentType,
		CdnAudioPlaylistType,
		CdnMp4Type,
	}

	// RTMP Connection
	StreamCreatePath          = "rtmpconns/create"
	StreamGetPath             = "rtmpconns/get/"
	StreamListPath            = "rtmpconns/list"
	StreamDeleteStreamIdPath  = "rtmpconns/streamId"
	StreamDeleteStreamKeyPath = "rtmpconns/streamKey"
	StreamKickPath            = "rtmpconns/kick/"

	// HLS Muxer
	StreamGetHLSMuxersListPath = "hlsmuxers/list"
	StreamGetHLSMuxerPath      = "hlsmuxers/get/"

	// Record
	StreamGetRecordPath    = "recordings/get/"
	StreamDeleteRecordPath = "recordings/deletesegment"

	StreamConnectPath    = "connect"
	StreamDisconnectPath = "disconnect"

	MaxRetries     = 5
	InitialBackoff = 500 * time.Millisecond
	MaxBackoff     = 2 * time.Second

	MinBalance = decimal.New(1, 18)

	TsFileType  = "ts"
	Mp4FileType = "mp4"
)
