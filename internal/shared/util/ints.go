package util

import (
	"fmt"
	"math"
)

func SafeInt64To32(val int64) (int32, error) {
	if val > math.MaxInt32 || val < math.MinInt32 {
		return 0, fmt.Errorf("value %d out of int32 range", val)
	}
	return int32(val), nil
}
