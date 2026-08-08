package utils

import "math"

// Saturating int conversions for narrowing casts flagged by gosec G115.
// Callers pass values that are already validated (pagination params, config
// values, len() results); clamping is a defense-in-depth guard, not a
// substitute for input validation.

// ClampToInt32 converts an int to int32, saturating at the type bounds
// instead of silently overflowing.
func ClampToInt32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

// ClampToInt16 converts an int to int16, saturating at the type bounds.
func ClampToInt16(v int) int16 {
	if v > math.MaxInt16 {
		return math.MaxInt16
	}
	if v < math.MinInt16 {
		return math.MinInt16
	}
	return int16(v)
}

// ClampInt64ToInt32 converts an int64 to int32, saturating at the type bounds.
func ClampInt64ToInt32(v int64) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

// ClampToUint32 converts a non-negative int to uint32, saturating at the
// bounds; negative input clamps to 0.
func ClampToUint32(v int) uint32 {
	if v < 0 {
		return 0
	}
	if uint64(v) > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(v)
}
