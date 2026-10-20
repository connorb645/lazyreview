package languages

import (
	"regexp"
	"strings"
)

var (
	headerAttribute = regexp.MustCompile(`@[\w.]+(?:\([^()]*\))?`)
	headerSoftWord  = regexp.MustCompile(`\b(?:any|some|out|in|public|private|protected|internal|final|abstract)\b`)
	headerMark      = regexp.MustCompile(`[~?!]`)
	headerCut       = jsRegexp(`[<(\[\s{]`)
	bareName        = regexp.MustCompile(`^[A-Za-z_$][\w$]*$`)
	identifier      = regexp.MustCompile(`[A-Za-z_$][\w$]*`)
)

// Identifiers is every identifier in a piece of code, in the order each
// first appears, once each.
func Identifiers(text string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, word := range identifier.FindAllString(text, -1) {
		if !seen[word] {
			seen[word] = true
			out = append(out, word)
		}
	}
	return out
}

func skipBalanced(text string, k int) int {
	if k >= len(text) {
		return len(text)
	}
	open := text[k]
	close, ok := map[byte]byte{'<': '>', '(': ')', '[': ']', '{': '}'}[open]
	depth := 0
	for j := k; j < len(text); j++ {
		switch c := text[j]; {
		case c == open:
			depth++
		case ok && c == close:
			if open == '<' && j > 0 && text[j-1] == '-' {
				continue
			}
			depth--
			if depth == 0 {
				return j + 1
			}
		}
	}
	return len(text)
}

func splitTopLevel(text string) []string {
	var parts []string
	depth, from := 0, 0
	for j := 0; j < len(text); j++ {
		switch c := text[j]; {
		case c == '<' || c == '(' || c == '[' || c == '{':
			depth++
		case (c == '>' && (j == 0 || text[j-1] != '-')) || c == ')' || c == ']' || c == '}':
			depth = max(0, depth-1)
		case c == ',' && depth == 0:
			parts = append(parts, text[from:j])
			from = j + 1
		}
	}
	parts = append(parts, text[from:])
	out := []string{}
	for _, part := range parts {
		if part = jsTrim(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func bareTypeName(entry string) string {
	text := headerAttribute.ReplaceAllString(entry, " ")
	text = headerSoftWord.ReplaceAllString(text, " ")
	text = jsTrim(headerMark.ReplaceAllString(text, " "))
	if cut := headerCut.FindStringIndex(text); cut != nil {
		text = text[:cut[0]]
	}
	name := ""
	for _, part := range strings.Split(text, ".") {
		if part != "" {
			name = part
		}
	}
	if !bareName.MatchString(name) {
		return ""
	}
	return name
}

func bareNames(entries []string) []string {
	out := []string{}
	for _, entry := range entries {
		if name := bareTypeName(entry); name != "" {
			out = append(out, name)
		}
	}
	return out
}
