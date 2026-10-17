package gui

import (
	"fmt"
	"regexp"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var inlineSVG = regexp.MustCompile(`(?is)<svg\b.*?</svg>`)

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

func withoutDiagrams(markdown string) string {
	return inlineSVG.ReplaceAllString(markdown, "\n\n*A diagram is here that a terminal cannot draw. gX opens this review in the web app.*\n\n")
}

func clipped(text string, width, rows int) []string {
	if r := []rune(text); len(r) > preview {
		text = string(r[:preview]) + "…"
	}
	lines := utils.Wrap(text, width)
	if len(lines) > rows {
		lines = lines[:rows]
		lines[rows-1] = ansi.Truncate(lines[rows-1]+" …", width, "…")
	}
	return lines
}
