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

// Around is n wrapped round into 0 to size-1, so it goes round at either
// end.
func Around(n, size int) int {
	return (n%size + size) % size
}
