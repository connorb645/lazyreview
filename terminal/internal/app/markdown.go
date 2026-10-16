package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	glamourstyle "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type markdown struct {
	style     glamourstyle.StyleConfig
	renderers map[int]*glamour.TermRenderer
	drawn     map[string][]string
}

func newMarkdown(dark bool) *markdown {
	style := styles.LightStyleConfig
	switch {
	case lipgloss.ColorProfile() == termenv.Ascii:
		style = styles.NoTTYStyleConfig
	case dark:
		style = styles.DarkStyleConfig
	}
	style.Code.Prefix, style.Code.Suffix = "", ""
	return &markdown{style: style, renderers: map[int]*glamour.TermRenderer{}, drawn: map[string][]string{}}
}

func (m *markdown) lines(text string, width int) []string {
	key := fmt.Sprintf("%d\x00%s", width, text)
	if drawn, ok := m.drawn[key]; ok {
		return drawn
	}
	renderer, ok := m.renderers[width]
	if !ok {
		var err error
		renderer, err = glamour.NewTermRenderer(glamour.WithStyles(m.style), glamour.WithColorProfile(lipgloss.ColorProfile()), glamour.WithWordWrap(width))
		if err != nil {
			return wrap(text, width)
		}
		m.renderers[width] = renderer
	}
	out, err := renderer.Render(text)
	if err != nil {
		return wrap(text, width)
	}
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		lines = append(lines, strings.TrimRight(l, " "))
	}
	for len(lines) > 0 && strings.TrimSpace(stripped(lines[0])) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(stripped(lines[len(lines)-1])) == "" {
		lines = lines[:len(lines)-1]
	}
	m.drawn[key] = lines
	return lines
}
