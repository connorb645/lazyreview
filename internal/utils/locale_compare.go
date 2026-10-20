package utils

import "strings"

const collationOrder = " _-,;:!?.'\"()[]{}@*/\\&#%`^+<=>|~$0123456789abcdefghijklmnopqrstuvwxyz"

// LocaleCompare orders a and b as JavaScript's localeCompare does in
// English, and says -1, 0 or 1: letters by the alphabet whatever their
// case, after digits, after punctuation in its own order; text that
// differs only in case puts lower case first. Other characters sort after
// the letters, by their code.
func LocaleCompare(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	for i := 0; i < len(ra) && i < len(rb); i++ {
		if pa, pb := collationKey(ra[i]), collationKey(rb[i]); pa != pb {
			return sign(pa - pb)
		}
	}
	if len(ra) != len(rb) {
		return sign(len(ra) - len(rb))
	}
	for i := range ra {
		if ra[i] != rb[i] {
			if isUpper(ra[i]) {
				return 1
			}
			return -1
		}
	}
	return 0
}

func collationKey(r rune) int {
	if isUpper(r) {
		r += 'a' - 'A'
	}
	if i := strings.IndexRune(collationOrder, r); i >= 0 {
		return i
	}
	return len(collationOrder) + int(r)
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}
