package number

import "math"

func Round(n float64, precision uint) float64 {
	if precision == 0 {
		return n
	}

	ratio := math.Pow(10, float64(precision))
	return math.Round(n*ratio) / ratio
}
