package numeric

import "math"

func Round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
