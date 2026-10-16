package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

type palette struct {
	Dark       bool
	Ink        lipgloss.Style
	Bold       lipgloss.Style
	Muted      lipgloss.Style
	Faint      lipgloss.Style
	Accent     lipgloss.Style
	Add        lipgloss.Style
	Del        lipgloss.Style
	Warn       lipgloss.Style
	Bad        lipgloss.Style
	Bar        lipgloss.Style
	Key        lipgloss.Style
	Border     lipgloss.Style
	Focus      lipgloss.Style
	pulls      [8]string
	components map[string]string
	plain      bool
	reset      string
	selected   string
	faded      string
}

func newPalette(dark bool) palette {
	pick := func(darkColour, lightColour string) lipgloss.Color {
		if dark {
			return lipgloss.Color(darkColour)
		}
		return lipgloss.Color(lightColour)
	}
	profile := lipgloss.ColorProfile()
	s := palette{Dark: dark, plain: profile == termenv.Ascii}
	s.Ink = lipgloss.NewStyle().Foreground(pick("#f1f0fb", "#17152f"))
	s.Bold = s.Ink.Bold(true)
	s.Muted = lipgloss.NewStyle().Foreground(pick("#a09dbf", "#615e7f"))
	s.Faint = lipgloss.NewStyle().Foreground(pick("#6d6a8a", "#9b98b5"))
	s.Accent = lipgloss.NewStyle().Foreground(pick("#a796ff", "#5233e6"))
	s.Add = lipgloss.NewStyle().Foreground(pick("#7ce2a7", "#146b3a"))
	s.Del = lipgloss.NewStyle().Foreground(pick("#ff9fb6", "#b21f43"))
	s.Warn = lipgloss.NewStyle().Foreground(pick("#f2b26a", "#9c5212"))
	s.Bad = s.Del.Bold(true)
	s.Bar = lipgloss.NewStyle().Foreground(pick("#a09dbf", "#615e7f"))
	s.Key = s.Accent.Bold(true)
	s.Border = lipgloss.NewStyle().Foreground(pick("#4a4766", "#c4c0dc"))
	s.Focus = s.Accent
	if !s.plain {
		s.reset = ansi.ResetStyle
		s.selected = background(profile, string(pick("#3a3270", "#dcd4ff")))
		s.faded = background(profile, string(pick("#26233d", "#efebfb")))
	}
	if dark {
		s.pulls = [8]string{"#3987e5", "#d95926", "#199e70", "#c98500", "#d55181", "#008300", "#9085e9", "#e66767"}
		s.components = map[string]string{"blue": "#7fa8f8", "teal": "#4cd3c2", "purple": "#b893fb", "amber": "#f2b26a", "olive": "#b8cc6a", "rose": "#f08cb6", "indigo": "#a4a9ff", "slate": "#a3afc8", "grey": "#9d9bb8"}
	} else {
		s.pulls = [8]string{"#2a78d6", "#eb6834", "#1baf7a", "#eda100", "#e87ba4", "#008300", "#6250d6", "#e34948"}
		s.components = map[string]string{"blue": "#2c66a8", "teal": "#0b6e6e", "purple": "#7448ad", "amber": "#9c5212", "olive": "#5f6e12", "rose": "#a8325e", "indigo": "#4a4fb0", "slate": "#556170", "grey": "#6f7987"}
	}
	return s
}

func background(profile termenv.Profile, colour string) string {
	return termenv.CSI + profile.Color(colour).Sequence(true) + "m"
}

func (s palette) pull(color int) lipgloss.Style {
	if color < 1 || color > 8 {
		return s.Muted
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.pulls[color-1]))
}

func (s palette) componentColor(color string) lipgloss.Style {
	if c, ok := s.components[color]; ok {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(c))
	}
	return s.Muted
}

func (s palette) highlight(line string, width int, active bool) string {
	line = fit(line, width)
	if s.plain {
		if active && strings.HasPrefix(line, " ") {
			return "▌" + line[1:]
		}
		return line
	}
	on := s.faded
	if active {
		on = s.selected
	}
	for _, reset := range []string{"\x1b[0m", "\x1b[m", "\x1b[49m"} {
		line = strings.ReplaceAll(line, reset, reset+on)
	}
	return on + line + s.reset
}
