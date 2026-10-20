package languages

import (
	"regexp"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

const jsSpaces = `\t\n\v\f\r \x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}`

func jsRegexp(pattern string) *regexp.Regexp {
	pattern = strings.ReplaceAll(pattern, `[\s\S]`, `(?s:.)`)
	var out strings.Builder
	inClass := false
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch {
		case c == '\\' && i+1 < len(pattern):
			i++
			switch next := pattern[i]; {
			case next == 's' && inClass:
				out.WriteString(jsSpaces)
			case next == 's':
				out.WriteString("[" + jsSpaces + "]")
			case next == 'S' && !inClass:
				out.WriteString("[^" + jsSpaces + "]")
			default:
				out.WriteByte(c)
				out.WriteByte(next)
			}
		case c == '[' && !inClass:
			inClass = true
			out.WriteByte(c)
		case c == ']' && inClass:
			inClass = false
			out.WriteByte(c)
		default:
			out.WriteByte(c)
		}
	}
	return regexp.MustCompile(out.String())
}

func group(text string, m []int, n int) string {
	if m[2*n] < 0 {
		return ""
	}
	return text[m[2*n]:m[2*n+1]]
}

func isJSSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0xa0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000, 0xfeff:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}

func jsTrim(s string) string {
	return strings.TrimFunc(s, isJSSpace)
}

func jsTrimStart(s string) string {
	return strings.TrimLeftFunc(s, isJSSpace)
}

func jsTrimEnd(s string) string {
	return strings.TrimRightFunc(s, isJSSpace)
}

func skipJSSpace(s string, j int) int {
	for j < len(s) {
		r, size := utf8.DecodeRuneInString(s[j:])
		if !isJSSpace(r) {
			break
		}
		j += size
	}
	return j
}

func leadingJSSpace(s string) int {
	count := 0
	for _, r := range s {
		if !isJSSpace(r) {
			break
		}
		count++
	}
	return count
}

func unitsOf(r rune) int {
	return max(1, utf16.RuneLen(r))
}

func unitsAhead(s string, from, units int) int {
	i := min(from, len(s))
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if unitsOf(r) > units {
			break
		}
		units -= unitsOf(r)
		i += size
	}
	return i
}

func unitsBack(s string, to, units int) int {
	i := to
	for i > 0 {
		r, size := utf8.DecodeLastRuneInString(s[:i])
		if unitsOf(r) > units {
			break
		}
		units -= unitsOf(r)
		i -= size
	}
	return i
}

func jsSlice(s string, from, to int) string {
	from, to = min(max(from, 0), len(s)), min(max(to, 0), len(s))
	if to < from {
		return ""
	}
	return s[from:to]
}

func indexFrom(s, sub string, from int) int {
	from = max(from, 0)
	if from > len(s) {
		return -1
	}
	at := strings.Index(s[from:], sub)
	if at < 0 {
		return -1
	}
	return from + at
}
