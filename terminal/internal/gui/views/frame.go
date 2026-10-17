package views

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/theme"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// Framed draws a view: its rows in a frame w wide and h high, with a title
// and tabs on top, a footer on the bottom and a scroll thumb on the right
// for the lines shown. A frame one line high is a strip of its title. The
// focused one is in the active border's colour, its title bold.
func Framed(content []string, shown Extent, w, h int, title, tabs, footer string, focused bool, th theme.Theme) []string {
	if w < 2 || h < 1 {
		return nil
	}
	border, heading, foot := th.InactiveBorder, th.Text, th.Muted
	if focused {
		border, heading, foot = th.ActiveBorder, th.ActiveBorder.Bold(true), th.ActiveBorder
	}
	inner := w - 2
	title = ansi.Truncate(title, max(0, inner-2), "…")
	tabs = ansi.Truncate(tabs, max(0, inner-2-ansi.StringWidth(title)), "")
	if h == 1 {
		footer = ansi.Truncate(footer, max(0, inner-3-ansi.StringWidth(title)-ansi.StringWidth(tabs)), "")
		fill := max(0, inner-2-ansi.StringWidth(title)-ansi.StringWidth(tabs)-ansi.StringWidth(footer))
		return []string{border.Render("──") + heading.Render(title) + tabs + border.Render(strings.Repeat("─", fill)) + foot.Render(footer) + border.Render("──")}
	}
	footer = ansi.Truncate(footer, max(0, inner-2), "…")
	used := ansi.StringWidth(title) + ansi.StringWidth(tabs)
	top := border.Render("╭─") + heading.Render(title) + tabs + border.Render(strings.Repeat("─", max(0, inner-1-used))+"╮")
	if title == "" {
		top = border.Render("╭" + strings.Repeat("─", inner) + "╮")
	}
	bottom := border.Render("╰" + strings.Repeat("─", inner) + "╯")
	if footer != "" {
		bottom = border.Render("╰"+strings.Repeat("─", max(0, inner-1-ansi.StringWidth(footer)))) + foot.Render(footer) + border.Render("─╯")
	}
	out := []string{top}
	side, thumb := border.Render("│"), border.Render("▐")
	start, size := shown.Thumb(h - 2)
	for i := 0; i < h-2; i++ {
		text, right := "", side
		if i < len(content) {
			text = content[i]
		}
		if i >= start && i < start+size {
			right = thumb
		}
		out = append(out, side+utils.Fit(text, inner)+right)
	}
	return append(out, bottom)
}

// PanelTitle is a panel's name after its number, as the title shows it.
func PanelTitle(n int, name string) string {
	return "[" + utils.Itoa(n) + "]─" + name
}

// Overlay draws a popup's lines over the screen's from a column and a row.
func Overlay(base, box []string, x, y int, th theme.Theme) []string {
	out := append([]string{}, base...)
	for i, l := range box {
		at := y + i
		if at < 0 || at >= len(out) {
			continue
		}
		row := out[at]
		w := ansi.StringWidth(l)
		left := utils.Fit(ansi.Truncate(row, x, ""), x)
		right := ansi.TruncateLeft(row, x+w, "")
		out[at] = left + th.Reset + l + th.Reset + right
	}
	return out
}
