package home

import "strings"

// beforeByName says whether name a sorts before b as the home screen has
// always sorted repositories: their names in lower case, compared as
// people's text is, punctuation before digits before letters, and _ before
// - before . before /.
func beforeByName(a, b string) bool {
	ka, kb := collated(strings.ToLower(a)), collated(strings.ToLower(b))
	for i := 0; i < len(ka) && i < len(kb); i++ {
		if ka[i] != kb[i] {
			return ka[i] < kb[i]
		}
	}
	return len(ka) < len(kb)
}

func collated(s string) []int {
	const order = "_-./0123456789abcdefghijklmnopqrstuvwxyz"
	out := make([]int, 0, len(s))
	for _, r := range s {
		if i := strings.IndexRune(order, r); i >= 0 {
			out = append(out, i)
		} else {
			out = append(out, len(order)+int(r))
		}
	}
	return out
}
