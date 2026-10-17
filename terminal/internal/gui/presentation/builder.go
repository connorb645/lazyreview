package presentation

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

type builder struct {
	lines []Line
	items []Item
	width int
	tr    *i18n.TranslationSet
	th    theme.Theme
}

func (b *builder) text(texts ...string) {
	for _, t := range texts {
		b.lines = append(b.lines, Line{Text: t, Item: -1})
	}
}

func (b *builder) prose(indent string, style lipgloss.Style, text string) {
	for _, l := range utils.Wrap(text, b.width-len(indent)-1) {
		b.text(indent + style.Render(l))
	}
}

func (b *builder) parts(indent string, parts ...string) {
	sep := b.th.Muted.Render(" · ")
	line := ""
	for _, part := range parts {
		switch {
		case line == "":
			line = part
		case ansi.StringWidth(indent+line+sep+part) <= b.width-1:
			line += sep + part
		default:
			b.text(indent + line)
			line = part
		}
	}
	b.text(indent + line)
}

func (b *builder) add(it Item, texts ...string) {
	b.items = append(b.items, it)
	for _, t := range texts {
		b.lines = append(b.lines, Line{Text: t, Item: len(b.items) - 1})
	}
}
