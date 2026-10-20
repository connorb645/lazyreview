package page

import (
	"strings"
	"unicode/utf16"

	"github.com/connorb645/lazyreview/terminal/internal/build/diff"
)

const (
	fileLimit  = 1_500_000
	totalLimit = 8_000_000
)

// ChangedTexts is each changed or renamed file's text at the head, as
// lines, by path, leaving out any file too large to carry, or that would
// take the page past its budget, and the paths it left out. Added files
// need none: the diff holds their whole text.
func ChangedTexts(files []diff.File) (map[string][]string, []string) {
	texts := map[string][]string{}
	var left []string
	total := 0
	for _, f := range files {
		if (f.Status != "M" && f.Status != "R") || f.Binary || f.Text == nil {
			continue
		}
		size := len(utf16.Encode([]rune(*f.Text)))
		if size > fileLimit || total+size > totalLimit {
			left = append(left, f.Path)
			continue
		}
		texts[f.Path] = Lines(*f.Text)
		total += size
	}
	return texts, left
}

// Lines is text as lines, without the empty line its final new line would
// leave.
func Lines(text string) []string {
	out := strings.Split(text, "\n")
	if out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}
