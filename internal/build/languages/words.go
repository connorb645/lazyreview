package languages

import "strings"

func isWordByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func wordAt(text, word string, from int) int {
	for i := max(from, 0); i <= len(text); {
		at := strings.Index(text[i:], word)
		if at < 0 {
			return -1
		}
		at += i
		end := at + len(word)
		if (at == 0 || !isWordByte(text[at-1])) && (end == len(text) || !isWordByte(text[end])) {
			return at
		}
		i = at + 1
	}
	return -1
}

func hasWord(text string, words ...string) bool {
	for _, word := range words {
		if wordAt(text, word, 0) >= 0 {
			return true
		}
	}
	return false
}
