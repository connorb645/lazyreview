package utils

// CopyBools is a copy of a set of flags, empty for none, so the copy can
// be changed without changing the set it came from.
func CopyBools(m map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, val := range m {
		out[k] = val
	}
	return out
}
