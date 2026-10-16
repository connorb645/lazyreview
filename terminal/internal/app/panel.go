package app

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// panel is a list drawn over the foot of a view, such as the keys the view
// takes: its rows, the row the cursor is on and the first row shown.
type panel struct {
	title  string
	rows   []panelRow
	cursor int
	top    int
}

func (pn panel) move(by, height int) panel {
	if len(pn.rows) == 0 {
		return pn
	}
	pn.cursor = clamp(pn.cursor+by, 0, len(pn.rows)-1)
	if pn.cursor < pn.top {
		pn.top = pn.cursor
	}
	if pn.cursor >= pn.top+height {
		pn.top = pn.cursor - height + 1
	}
	return pn
}

// lines draws the panel in a frame width cells wide and at most height rows high.
func (pn panel) lines(width, height int, p palette) []string {
	rows := min(len(pn.rows), max(1, height-3))
	left := 0
	for _, r := range pn.rows {
		left = max(left, ansi.StringWidth(r.left))
	}
	inner := width - 4
	out := []string{p.Muted.Render("╭" + strings.Repeat("─", width-2) + "╮")}
	frame := func(text string) string {
		return p.Muted.Render("│ ") + fit(text, inner) + p.Muted.Render(" │")
	}
	out = append(out, frame(p.Bold.Render(pn.title)))
	for i := pn.top; i < pn.top+rows && i < len(pn.rows); i++ {
		r := pn.rows[i]
		text := p.Key.Render(r.left) + strings.Repeat(" ", left-ansi.StringWidth(r.left)+2) + p.Ink.Render(r.right)
		row := frame(text)
		if i == pn.cursor && pn.rows[i].open != nil {
			row = p.Muted.Render("│ ") + p.Selected.Render(fit(text, inner)) + p.Muted.Render(" │")
		}
		out = append(out, row)
	}
	out = append(out, p.Muted.Render("╰"+strings.Repeat("─", width-2)+"╯"))
	return out
}
