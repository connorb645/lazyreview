package layout

import "strconv"

func grouped(n int) string {
	text := strconv.Itoa(n)
	sign := ""
	if n < 0 {
		sign, text = "-", text[1:]
	}
	for i := len(text) - 3; i > 0; i -= 3 {
		text = text[:i] + "," + text[i:]
	}
	return sign + text
}
