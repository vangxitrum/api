package file_range

import (
	"fmt"
	"strconv"
	"strings"
)

func GetFileSizeFromRange(fileRange string) (int64, error) {
	positions := strings.Split(fileRange, ",")

	if len(positions) != 2 {
		return 0, fmt.Errorf("invalid format")
	}

	size, err := strconv.ParseInt(positions[1], 10, 64)

	if err != nil {
		return 0, err
	}
	return size, nil
}
