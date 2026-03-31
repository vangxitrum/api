package utils

import (
	"time"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
)

func GetTimeRangeHour(now time.Time, hoursOffset int) models.TimeRange {
	start := now.Truncate(time.Hour).Add(time.Duration(hoursOffset) * time.Hour)
	end := start.Add(time.Hour)
	return models.TimeRange{Start: start, End: end}
}
