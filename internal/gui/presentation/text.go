package presentation

import (
	"regexp"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/utils"
)

var inlineSVG = regexp.MustCompile(`(?is)<svg\b.*?</svg>`)

// LinesWords names lines as a person would, such as "line 5", "lines 3-5"
// or "removed line 2 to line 4".
func LinesWords(tr *i18n.TranslationSet, side string, line, startLine int, startSide string) string {
	one := func(s string, n int) string {
		words := tr.LineAt
		if s == "LEFT" {
			words = tr.RemovedLineAt
		}
		return utils.ResolvePlaceholderString(words, map[string]string{"line": utils.Itoa(n)})
	}
	if startLine == 0 {
		return one(side, line)
	}
	if startSide == "" {
		startSide = side
	}
	if startSide != side {
		return utils.ResolvePlaceholderString(tr.LineToLine, map[string]string{"from": one(startSide, startLine), "to": one(side, line)})
	}
	words := tr.LinesAt
	if side == "LEFT" {
		words = tr.RemovedLinesAt
	}
	return utils.ResolvePlaceholderString(words, map[string]string{"from": utils.Itoa(startLine), "to": utils.Itoa(line)})
}

func withoutDiagrams(tr *i18n.TranslationSet, labels KeyLabels, markdown string) string {
	notDrawn := utils.ResolvePlaceholderString(tr.DrawingNotShown, map[string]string{"drawings": labels.Of("drawings")})
	return inlineSVG.ReplaceAllLiteralString(markdown, "\n\n*"+notDrawn+"*\n\n")
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
