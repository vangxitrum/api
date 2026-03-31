package core

const (
	ffmpegMediaStreamValue     = "0:v:0"
	ffmpegAudioStreamValue     = "0:a:0?"
	ffmpegMediaCodecValue      = "libx264"
	ffmpegAudioCodecValue      = "aac"
	ffmpegAudioSampleRateValue = "48000"
	ffmpegMediaFilterValue     = "scale=w=%d:%d:h=-2"
	ffmpegAudioBitrateValue    = "128k"
	ffmpegEncodingPresetValue  = "slow"
	ffmpegHLSListSizeValue     = "0"
	ffmpegOutputFormatValue    = "hls"
	ffmpegPlaylistTypeValue    = "event"
	ffmpegSegmentDurationValue = "6"
	ffmpegHLSFlagsValue        = "independent_segments"
	ffprobePath                = "./ffmpegd/ffprobe"
	ffmpegPath                 = "./ffmpegd/ffmpeg"
)
