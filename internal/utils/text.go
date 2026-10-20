// Package utils holds small helpers every layer of the terminal app
// shares: fitting a line or a title to a width, wrapping, counting words,
// filling a string's placeholders, keeping a number in a range.
package utils

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
)

const nonBreakingHyphen = "\u2011"

var (
	fencedCode = regexp.MustCompile("(?s)```.*?```")
	mdLink     = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	listMark   = regexp.MustCompile(`(?m)^\s*[-+]\s+`)
)

// Plural is a count and its word, such as "1 file" or "3 files".
func Plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	if strings.HasSuffix(word, "y") && !strings.HasSuffix(word, "ey") {
		return fmt.Sprintf("%d %sies", n, strings.TrimSuffix(word, "y"))
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// Short is a commit id cut to its first eight characters.
func Short(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// When is an RFC 3339 time as a date and a time in a time zone, such as
// "15 Oct, 14:00", or the text as it is when it is not a time.
func When(iso string, location *time.Location) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return t.In(location).Format("2 Jan, 15:04")
}

// Fit cuts a line to a width, with an ellipsis, or pads it to the width.
func Fit(line string, width int) string {
	if width <= 0 {
		return ""
	}
	cut := ansi.Truncate(line, width, "…")
	if pad := width - ansi.StringWidth(cut); pad > 0 {
		cut += strings.Repeat(" ", pad)
	}
	return cut
}

// FitTitle cuts a title to a width: a path loses its first folders, any
// other title its end.
func FitTitle(title string, width int) string {
	switch {
	case width <= 0:
		return ""
	case ansi.StringWidth(title) <= width:
		return title
	case !strings.Contains(title, "/"):
		return ansi.Truncate(title, width, "…")
	}
	parts := strings.Split(title, "/")
	for i := 1; i < len(parts); i++ {
		if tail := "…/" + strings.Join(parts[i:], "/"); ansi.StringWidth(tail) <= width {
			return tail
		}
	}
	return ansi.TruncateLeft(title, ansi.StringWidth(title)-width+1, "…")
}

// FitPathIn fits a path between words that stay whole, in a width.
func FitPathIn(head, path, tail string, width int) string {
	room := width - ansi.StringWidth(head) - ansi.StringWidth(tail)
	if room < 1 {
		return head + path + tail
	}
	return head + FitTitle(path, room) + tail
}

// Wrap breaks text into lines of a width, at spaces, never at a hyphen.
func Wrap(text string, width int) []string {
	if width < 8 {
		width = 8
	}
	var out []string
	for _, paragraph := range strings.Split(text, "\n") {
		wrapped := ansi.Wrap(strings.ReplaceAll(paragraph, "-", nonBreakingHyphen), width, " ")
		out = append(out, strings.Split(strings.ReplaceAll(wrapped, nonBreakingHyphen, "-"), "\n")...)
	}
	return out
}

// Itoa is a number as text.
func Itoa(n int) string {
	return fmt.Sprint(n)
}

// Stripped is a line without its colour and style codes.
func Stripped(line string) string {
	return ansi.Strip(line)
}

// PlainText is Markdown read as plain words, for a short preview.
func PlainText(markdown string) string {
	out := fencedCode.ReplaceAllString(markdown, " ")
	out = mdLink.ReplaceAllString(out, "$1")
	out = listMark.ReplaceAllString(out, "")
	out = strings.NewReplacer("`", "", "*", "", "_", "", ">", "", "#", "").Replace(out)
	return strings.Join(strings.Fields(out), " ")
}

// Spread puts words on the left and on the right of a width, cutting the
// left ones when both do not fit.
func Spread(left, right string, width int) string {
	gap := width - ansi.StringWidth(left) - ansi.StringWidth(right)
	if gap < 1 {
		return Fit(left, width)
	}
	return left + strings.Repeat(" ", gap) + right
}
