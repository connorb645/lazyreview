package app

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
)

var (
	inlineSVG  = regexp.MustCompile(`(?is)<svg\b.*?</svg>`)
	fencedCode = regexp.MustCompile("(?s)```.*?```")
	mdLink     = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	listMark   = regexp.MustCompile(`(?m)^\s*[-+]\s+`)
)

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	if strings.HasSuffix(word, "y") && !strings.HasSuffix(word, "ey") {
		return fmt.Sprintf("%d %sies", n, strings.TrimSuffix(word, "y"))
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func short(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// when is a time the server wrote, such as 15 Oct, 13:00, in the app's time zone.
func when(iso string, location *time.Location) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return t.In(location).Format("2 Jan, 15:04")
}

// fit cuts a line, styled or not, to width cells, and pads it out to them.
func fit(line string, width int) string {
	if width <= 0 {
		return ""
	}
	cut := ansi.Truncate(line, width, "…")
	if pad := width - ansi.StringWidth(cut); pad > 0 {
		cut += strings.Repeat(" ", pad)
	}
	return cut
}

// wrap breaks plain text into lines of at most width cells, at spaces where it can.
func wrap(text string, width int) []string {
	if width < 8 {
		width = 8
	}
	var out []string
	for _, paragraph := range strings.Split(text, "\n") {
		out = append(out, strings.Split(ansi.Wrap(paragraph, width, " "), "\n")...)
	}
	return out
}

func linesWords(side string, line, startLine int, startSide string) string {
	name := func(s string) string {
		if s == "LEFT" {
			return "removed line"
		}
		return "line"
	}
	if startLine == 0 {
		return fmt.Sprintf("%s %d", name(side), line)
	}
	if startSide == "" {
		startSide = side
	}
	if startSide != side {
		return fmt.Sprintf("%s %d to %s %d", name(startSide), startLine, name(side), line)
	}
	return fmt.Sprintf("%ss %d-%d", name(side), startLine, line)
}

func itoa(n int) string {
	return fmt.Sprint(n)
}

func withoutDiagrams(markdown string) string {
	return inlineSVG.ReplaceAllString(markdown, "\n\n*A diagram is here that a terminal cannot draw. gX opens this review in the web app.*\n\n")
}

func stripped(line string) string {
	return ansi.Strip(line)
}

// plainText is Markdown read as plain words, for a short preview.
func plainText(markdown string) string {
	out := fencedCode.ReplaceAllString(markdown, " ")
	out = mdLink.ReplaceAllString(out, "$1")
	out = listMark.ReplaceAllString(out, "")
	out = strings.NewReplacer("`", "", "*", "", "_", "", ">", "", "#", "").Replace(out)
	return strings.Join(strings.Fields(out), " ")
}

// clipped is text cut to `preview` characters and wrapped to width, at most `rows` lines.
func clipped(text string, width, rows int) []string {
	if r := []rune(text); len(r) > preview {
		text = string(r[:preview]) + "…"
	}
	lines := wrap(text, width)
	if len(lines) > rows {
		lines = lines[:rows]
		lines[rows-1] = ansi.Truncate(lines[rows-1]+" …", width, "…")
	}
	return lines
}
