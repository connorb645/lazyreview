package gui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

type menu struct {
	title   string
	rows    []menuRow
	cursor  int
	top     int
	finding bool
	query   string
	all     []menuRow
}

func (pn menu) narrowed(query string) menu {
	pn.query, pn.cursor, pn.top = query, 0, 0
	q := strings.ToLower(query)
	var holding, spread []menuRow
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

func (pn menu) move(by, height int) menu {
	if len(pn.rows) == 0 {
		return pn
	}
	at := pn.cursor
	for step := 0; step < abs(by); step++ {
		next := at + sign(by)
		for next >= 0 && next < len(pn.rows) && pn.rows[next].heading != "" {
			next += sign(by)
		}
		if next < 0 || next >= len(pn.rows) {
			break
		}
		at = next
	}
	pn.cursor = at
	if pn.cursor < pn.top {
		pn.top = pn.cursor
		for pn.top > 0 && pn.rows[pn.top-1].heading != "" {
			pn.top--
		}
	}
	if pn.cursor >= pn.top+height {
		pn.top = pn.cursor - height + 1
	}
	return pn
}

func (pn menu) first() menu {
	for i, r := range pn.rows {
		if r.heading == "" {
			pn.cursor = i
			return pn
		}
	}
	return pn
}

func (pn menu) choices() (int, int) {
	at, n := 0, 0
	for i, r := range pn.rows {
		if r.heading != "" {
			continue
		}
		n++
		if i <= pn.cursor {
			at = n
		}
	}
	return at, n
}

func (pn menu) box(width, height int, p palette) []string {
	left, right := 0, 0
	for _, r := range pn.rows {
		left = max(left, ansi.StringWidth(r.left))
		right = max(right, ansi.StringWidth(r.right))
	}
	w := min(width-4, max(left+right+7, width/2, 44))
	if !pn.choosing() {
		w = min(width-4, max(width*3/5, 60))
	}
	inner := w - 2
	rows := height - 2
	var content []string
	if pn.finding {
		rows--
		content = append(content, " "+p.Accent.Render("›")+" "+p.Ink.Render(pn.query)+p.Accent.Render("▏"))
	}
	if pn.finding && len(pn.rows) == 0 {
		content = append(content, " "+p.Muted.Render("nothing matches"))
	}
	drawn := make([][]string, len(pn.rows))
	for i, r := range pn.rows {
		drawn[i] = pn.drawRow(r, i == pn.cursor && pn.choosing(), left, inner, p)
	}
	top := min(pn.top, pn.cursor)
	for top < pn.cursor && linesIn(drawn, top, pn.cursor) > rows {
		top++
	}
	for i := top; i < len(drawn) && len(content) < height-2; i++ {
		for _, l := range drawn[i] {
			if len(content) < height-2 {
				content = append(content, l)
			}
		}
	}
	footer := ""
	if at, n := pn.choices(); n > 0 && pn.choosing() {
		footer = fmt.Sprintf("%d of %d", at, n)
	}
	h := len(content) + 2
	if pn.finding {
		h = height
	}
	return framed(content, extent{}, w, h, pn.title, "", footer, true, p)
}

func (pn menu) drawRow(r menuRow, selected bool, left, inner int, p palette) []string {
	if r.heading != "" {
		return []string{" " + p.Focus.Bold(true).Render(r.heading)}
	}
	gap := 0
	if left > 0 {
		gap = left + 2
	}
	room := max(8, inner-gap-2)
	words := []string{r.right}
	if !pn.finding {
		words = utils.Wrap(r.right, room)
	}
	var out []string
	for i, text := range words {
		key := ""
		if i == 0 {
			key = r.left
		}
		fitted := ansi.Truncate(text, room, "…")
		if pn.finding {
			fitted = utils.FitTitle(text, room)
		}
		line := " " + p.Key.Render(key) + strings.Repeat(" ", max(0, gap-ansi.StringWidth(key))) + p.Ink.Render(fitted)
		if selected {
			line = p.highlight(line, inner, true)
		}
		out = append(out, line)
	}
	return out
}

func linesIn(drawn [][]string, from, to int) int {
	n := 0
	for i := from; i <= to && i < len(drawn); i++ {
		n += len(drawn[i])
	}
	return n
}

func (pn menu) choosing() bool {
	for _, r := range pn.rows {
		if r.open != nil || r.left != "" {
			return true
		}
	}
	return pn.finding
}
