package context

import (
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

// MenuContext is a popup list: the menu, the keys, the finder with every
// row it narrows from, or posting's answer.
type MenuContext struct {
	presentation.Menu
	All []presentation.MenuRow
}

// Narrowed is the finder's rows that hold what is typed, those with its
// letters in order after them.
func (pn MenuContext) Narrowed(query string) MenuContext {
	pn.Query, pn.Cursor, pn.Top = query, 0, 0
	q := strings.ToLower(query)
	var holding, spread []presentation.MenuRow
	for _, r := range pn.All {
		text := strings.ToLower(r.Right)
		switch {
		case strings.Contains(text, q):
			holding = append(holding, r)
		case inOrder(text, q):
			spread = append(spread, r)
		}
	}
	pn.Rows = append(holding, spread...)
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

// Move is the menu with its cursor moved by rows, past headings, shown in a
// height.
func (pn MenuContext) Move(by, height int) MenuContext {
	if len(pn.Rows) == 0 {
		return pn
	}
	at := pn.Cursor
	for step := 0; step < max(by, -by); step++ {
		next := at + sign(by)
		for next >= 0 && next < len(pn.Rows) && pn.Rows[next].Heading != "" {
			next += sign(by)
		}
		if next < 0 || next >= len(pn.Rows) {
			break
		}
		at = next
	}
	pn.Cursor = at
	if pn.Cursor < pn.Top {
		pn.Top = pn.Cursor
		for pn.Top > 0 && pn.Rows[pn.Top-1].Heading != "" {
			pn.Top--
		}
	}
	if pn.Cursor >= pn.Top+height {
		pn.Top = pn.Cursor - height + 1
	}
	return pn
}

// First is the menu with its cursor on its first row that is not a heading.
func (pn MenuContext) First() MenuContext {
	for i, r := range pn.Rows {
		if r.Heading == "" {
			pn.Cursor = i
			return pn
		}
	}
	return pn
}
