package gui

import (
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

type menu struct {
	presentation.Menu
	all []presentation.MenuRow
}

func (pn menu) narrowed(query string) menu {
	pn.Query, pn.Cursor, pn.Top = query, 0, 0
	q := strings.ToLower(query)
	var holding, spread []presentation.MenuRow
	for _, r := range pn.all {
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

func (pn menu) move(by, height int) menu {
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

func (pn menu) first() menu {
	for i, r := range pn.Rows {
		if r.Heading == "" {
			pn.Cursor = i
			return pn
		}
	}
	return pn
}
