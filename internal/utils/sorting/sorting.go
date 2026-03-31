package sorting

import (
	"sort"
	"strconv"
	"strings"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
)

func SortQualities(qualities []*models.MediaQuality) string {
	qualitiesString := make([]string, 0, len(qualities))
	for _, q := range qualities {
		qualitiesString = append(qualitiesString, q.Resolution)
	}

	sort.Slice(qualitiesString, func(i, j int) bool {
		if qualitiesString[i] == "" {
			return false
		}

		if qualitiesString[j] == "" {
			return true
		}

		x, err := strconv.Atoi(strings.TrimSuffix(qualitiesString[i], "p"))
		if err != nil {
			return false
		}

		y, err := strconv.Atoi(strings.TrimSuffix(qualitiesString[j], "p"))
		if err != nil {
			return true
		}

		return x > y
	})
	return strings.Join(qualitiesString, ",")
}
