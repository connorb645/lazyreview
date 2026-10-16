package app

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// panel is a list drawn over the foot of a view, such as the keys the view
// takes: its rows, the row the cursor is on and the first row shown. A
// finder narrows every row it holds to those that match what is typed.
type panel struct {
	title   string
	rows    []panelRow
	cursor  int
	top     int
	finding bool
	query   string
	all     []panelRow
}

// narrowed is the finder with only the rows that match query: those that
// hold it first, then those that hold its letters in order.
func (pn panel) narrowed(query string) panel {
	pn.query, pn.cursor, pn.top = query, 0, 0
	q := strings.ToLower(query)
	var holding, spread []panelRow
	for _, r := range pn.all {
		text := strings.ToLower(r.right)
		switch {
		case strings.Contains(text, q):
			holding = append(holding, r)
		case inOrder(text, q):
			spread = append(spread, r)
		}
	}
	pn.rows = append(holding, spread...)
	return pn
}

func inOrder(text, letters string) bool {
	for _, r := range letters {
		i := strings.IndexRune(text, r)
		if i < 0 {
			return false
		}
		text = text[i+len(string(r)):]
	}
	return true
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
	if pn.finding {
		rows = max(1, height-3)
	}
	left := 0
	for _, r := range pn.rows {
		left = max(left, ansi.StringWidth(r.left))
	}
	inner := width - 4
	out := []string{p.Muted.Render("╭" + strings.Repeat("─", width-2) + "╮")}
	frame := func(text string) string {
		return p.Muted.Render("│ ") + fit(text, inner) + p.Muted.Render(" │")
	}
	title := p.Bold.Render(pn.title)
	if pn.finding {
		title += "  " + p.Ink.Render(pn.query) + p.Accent.Render("▏")
	}
	out = append(out, frame(title))
	if pn.finding && len(pn.rows) == 0 {
		out = append(out, frame(p.Muted.Render("nothing matches")))
	}
	for i := pn.top; i < pn.top+rows && i < len(pn.rows); i++ {
		r := pn.rows[i]
		text := p.Key.Render(r.left) + strings.Repeat(" ", left-ansi.StringWidth(r.left)+2) + p.Ink.Render(r.right)
		row := frame(text)
		if i == pn.cursor && pn.rows[i].open != nil {
			row = p.Muted.Render("│ ") + p.Selected.Render(fit(text, inner)) + p.Muted.Render(" │")
		}
		out = append(out, row)
	}
	for len(out) < rows+2 {
		out = append(out, frame(""))
	}
	out = append(out, p.Muted.Render("╰"+strings.Repeat("─", width-2)+"╯"))
	return out
}
