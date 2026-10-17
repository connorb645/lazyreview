// Package theme turns the terminal app's colours into the styles it draws
// with, for the colours the terminal can show.
package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// Theme is the styles the terminal app draws with, made from its colours
// for a dark or a light terminal and the colours the terminal can show.
// Keys and the focused title are bold. Plain is a terminal of no colour,
// which marks the selected line in its first cell instead.
type Theme struct {
	Colours                    Colours
	Dark                       bool
	Plain                      bool
	Text                       lipgloss.Style
	Bold                       lipgloss.Style
	Muted                      lipgloss.Style
	Faint                      lipgloss.Style
	Accent                     lipgloss.Style
	Added                      lipgloss.Style
	Removed                    lipgloss.Style
	Warning                    lipgloss.Style
	Bad                        lipgloss.Style
	Key                        lipgloss.Style
	InactiveBorder             lipgloss.Style
	ActiveBorder               lipgloss.Style
	Reset                      string
	SelectedLineBg             string
	InactiveViewSelectedLineBg string
	VisualSelectionBg          string
}

// New makes the theme of colours for a dark or a light terminal that shows
// the colours of profile.
func New(colours Colours, dark bool, profile termenv.Profile) Theme {
	fg := func(colour string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(colour)) }
	t := Theme{Colours: colours, Dark: dark, Plain: profile == termenv.Ascii}
	t.Text = fg(colours.DefaultFgColor)
	t.Bold = t.Text.Bold(true)
	t.Muted = fg(colours.MutedFgColor)
	t.Faint = fg(colours.FaintFgColor)
	t.Accent = fg(colours.AccentColor)
	t.Added = fg(colours.AddedColor)
	t.Removed = fg(colours.RemovedColor)
	t.Warning = fg(colours.WarningColor)
	t.Bad = t.Removed.Bold(true)
	t.Key = t.Accent.Bold(true)
	t.InactiveBorder = fg(colours.InactiveBorderColor)
	t.ActiveBorder = fg(colours.ActiveBorderColor)
	if !t.Plain {
		t.Reset = ansi.ResetStyle
		t.SelectedLineBg = background(profile, colours.SelectedLineBgColor)
		t.InactiveViewSelectedLineBg = background(profile, colours.InactiveViewSelectedLineBgColor)
		t.VisualSelectionBg = background(profile, colours.VisualSelectionBgColor)
	}
	return t
}

func background(profile termenv.Profile, colour string) string {
	return termenv.CSI + profile.Color(colour).Sequence(true) + "m"
}

// PullRequest is the style of a stack's pull request by its colour number,
// from 1, or Muted for a number with no colour.
func (t Theme) PullRequest(color int) lipgloss.Style {
	if color < 1 || color > len(t.Colours.PullRequestColors) {
		return t.Muted
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(t.Colours.PullRequestColors[color-1]))
}

// Component is the style of a component by its colour's name, or Muted for
// a name with no colour.
func (t Theme) Component(color string) lipgloss.Style {
	if c, ok := t.Colours.ComponentColors[color]; ok {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(c))
	}
	return t.Muted
}

// Selected draws a line as the selected one, fitted to a width: on the
// selected line's background where the keys are, and on the inactive one
// elsewhere.
func (t Theme) Selected(line string, width int, active bool) string {
	line = utils.Fit(line, width)
	if t.Plain {
		if active && strings.HasPrefix(line, " ") {
			return "▌" + line[1:]
		}
		return line
	}
	if active {
		return t.under(line, t.SelectedLineBg)
	}
	return t.under(line, t.InactiveViewSelectedLineBg)
}

// Visual draws a line as one of a visual selection, fitted to a width.
func (t Theme) Visual(line string, width int) string {
	line = utils.Fit(line, width)
	if t.Plain {
		if strings.HasPrefix(line, " ") {
			return "┃" + line[1:]
		}
		return line
	}
	return t.under(line, t.VisualSelectionBg)
}

func (t Theme) under(line, on string) string {
	for _, reset := range []string{"\x1b[0m", "\x1b[m", "\x1b[49m"} {
		line = strings.ReplaceAll(line, reset, reset+on)
	}
	return on + line + t.Reset
}
