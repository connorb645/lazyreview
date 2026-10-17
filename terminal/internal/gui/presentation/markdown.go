// Package presentation turns the review's data and the gui's state into the
// styled rows the views draw.
package presentation

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	glamourstyle "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// Markdown draws Markdown for a terminal, keeping a renderer for each width
// and each text it has drawn.
type Markdown struct {
	style     glamourstyle.StyleConfig
	renderers map[int]*glamour.TermRenderer
	drawn     map[string][]string
}

// NewMarkdown returns a Markdown that draws for a dark terminal or a light
// one, in the colours the terminal can show.
func NewMarkdown(dark bool) *Markdown {
	style := styles.LightStyleConfig
	switch {
	case lipgloss.ColorProfile() == termenv.Ascii:
		style = styles.NoTTYStyleConfig
	case dark:
		style = styles.DarkStyleConfig
	}
	style.Code.Prefix, style.Code.Suffix = "", ""
	return &Markdown{style: style, renderers: map[int]*glamour.TermRenderer{}, drawn: map[string][]string{}}
}

// Lines is text drawn as Markdown in a width, without the blank lines
// round it.
func (m *Markdown) Lines(text string, width int) []string {
	key := fmt.Sprintf("%d\x00%s", width, text)
	if drawn, ok := m.drawn[key]; ok {
		return drawn
	}
	renderer, ok := m.renderers[width]
	if !ok {
		var err error
		renderer, err = glamour.NewTermRenderer(glamour.WithStyles(m.style), glamour.WithColorProfile(lipgloss.ColorProfile()), glamour.WithWordWrap(width))
		if err != nil {
			return utils.Wrap(text, width)
		}
		m.renderers[width] = renderer
	}
	out, err := renderer.Render(text)
	if err != nil {
		return utils.Wrap(text, width)
	}
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		lines = append(lines, strings.TrimRight(l, " "))
	}
	for len(lines) > 0 && strings.TrimSpace(utils.Stripped(lines[0])) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(utils.Stripped(lines[len(lines)-1])) == "" {
		lines = lines[:len(lines)-1]
	}
	m.drawn[key] = lines
	return lines
}
