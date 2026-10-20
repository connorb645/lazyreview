package placement

import (
	"regexp"
	"strings"
	"unicode/utf16"
)

// Pattern is a path pattern from the config: ** matches across folders, *
// within one, ? one character. A pattern with no wildcard matches that path
// and everything under it, and one ending in / matches everything under
// it. The more literal characters it holds, the more specific it is, and an
// exact path is the most specific of all.
type Pattern struct {
	Text        string
	Specificity float64
	exact       string
	folder      string
	pattern     *regexp.Regexp
}

// NewPattern reads the pattern text.
func NewPattern(text string) Pattern {
	if !strings.ContainsAny(text, "*?") {
		folder := text
		if !strings.HasSuffix(folder, "/") {
			folder += "/"
		}
		specificity := float64(jsLength(text))
		if !strings.HasSuffix(text, "/") {
			specificity += 0.5
		}
		return Pattern{Text: text, Specificity: specificity, exact: strings.TrimSuffix(text, "/"), folder: folder}
	}
	var source strings.Builder
	for i := 0; i < len(text); i++ {
		switch {
		case text[i] == '*' && i+1 < len(text) && text[i+1] == '*':
			if i+2 < len(text) && text[i+2] == '/' {
				source.WriteString("(?:.*/)?")
				i += 2
			} else {
				source.WriteString(".*")
				i++
			}
		case text[i] == '*':
			source.WriteString("[^/]*")
		case text[i] == '?':
			source.WriteString("[^/]")
		default:
			source.WriteString(regexp.QuoteMeta(text[i : i+1]))
		}
	}
	if strings.HasSuffix(text, "/") {
		source.WriteString(".*")
	}
	literal := strings.NewReplacer("*", "", "?", "").Replace(text)
	return Pattern{Text: text, Specificity: float64(jsLength(literal)), pattern: regexp.MustCompile("^" + source.String() + "$")}
}

// Matches says whether the pattern matches path.
func (p Pattern) Matches(path string) bool {
	if p.pattern == nil {
		return path == p.exact || strings.HasPrefix(path, p.folder)
	}
	return p.pattern.MatchString(path)
}

func jsLength(text string) int {
	return len(utf16.Encode([]rune(text)))
}
