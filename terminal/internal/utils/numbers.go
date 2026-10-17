package utils

// Clamp keeps a number from low to high.
func Clamp(n, low, high int) int {
	if n < low {
		return low
	}
	if n > high {
		return high
	}
	return n
}
