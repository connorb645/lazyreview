package app

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func framed(content []string, shown extent, w, h int, title, tabs, footer string, focused bool, p palette) []string {
	if w < 2 || h < 2 {
		return nil
	}
	border, heading, foot := p.Border, p.Ink, p.Muted
	if focused {
		border, heading, foot = p.Focus, p.Focus.Bold(true), p.Focus
	}
	inner := w - 2
	title = ansi.Truncate(title, max(0, inner-2), "…")
	tabs = ansi.Truncate(tabs, max(0, inner-2-ansi.StringWidth(title)), "")
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
	start, size := shown.thumb(h - 2)
	for i := 0; i < h-2; i++ {
		text, right := "", side
		if i < len(content) {
			text = content[i]
		}
		if i >= start && i < start+size {
			right = thumb
		}
		out = append(out, side+fit(text, inner)+right)
	}
	return append(out, bottom)
}

func panelTitle(n int, name string) string {
	return "[" + itoa(n) + "]─" + name
}

func overlay(base, box []string, x, y int, p palette) []string {
	out := append([]string{}, base...)
	for i, l := range box {
		at := y + i
		if at < 0 || at >= len(out) {
			continue
		}
		row := out[at]
		w := ansi.StringWidth(l)
		left := fit(ansi.Truncate(row, x, ""), x)
		right := ansi.TruncateLeft(row, x+w, "")
		out[at] = left + p.reset + l + p.reset + right
	}
	return out
}
