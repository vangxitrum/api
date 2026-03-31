package convert

func ToKbs(bytes int64) int64 {
	if bytes < 0 {
		return 0
	}
	return bytes / 1000
}
