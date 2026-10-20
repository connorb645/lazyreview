package context

import (
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/utils"
)

// SideContext is a panel on the left: its rows, the selected one, which is
// never a heading, and the first one shown.
type SideContext struct {
	Rows   []presentation.SideRow
	Cursor int
	Top    int
}

// Reset is the panel with new rows, the same row selected when it is still
// there, or the nearest one.
func (s SideContext) Reset(rows []presentation.SideRow) SideContext {
	next := SideContext{Rows: rows, Top: s.Top, Cursor: -1}
	if was, ok := s.Selected(); ok {
		next.Cursor = next.Index(was.Key)
	}
	if next.Cursor < 0 {
		next.Cursor = next.Landing(min(s.Cursor, len(rows)-1), 1)
	}
	return next
}

// Selected is the selected row, if there is one.
func (s SideContext) Selected() (presentation.SideRow, bool) {
	if s.Cursor < 0 || s.Cursor >= len(s.Rows) || s.Rows[s.Cursor].Heading {
		return presentation.SideRow{}, false
	}
	return s.Rows[s.Cursor], true
}

// Index is the position of the row with a key, or -1.
func (s SideContext) Index(key string) int {
	for i, r := range s.Rows {
		if r.Key == key && !r.Heading {
			return i
		}
	}
	return -1
}

// Landing is the first row from one that is not a heading, going by a
// step, then back the other way, or -1.
func (s SideContext) Landing(from, by int) int {
	for i := max(0, from); i >= 0 && i < len(s.Rows); i += by {
		if !s.Rows[i].Heading {
			return i
		}
	}
	for i := min(from, len(s.Rows)-1); i >= 0 && i < len(s.Rows); i -= by {
		if !s.Rows[i].Heading {
			return i
		}
	}
	return -1
}

// Move is the panel with its selection moved by rows, past headings, the
// selection shown in a height.
func (s SideContext) Move(by, height int) SideContext {
	if s.Cursor < 0 {
		return s
	}
	at := s.Cursor
	for step := by; step != 0; {
		next := s.Landing(at+sign(step), sign(step))
		if next < 0 || next == at || (next-at)*sign(step) < 0 {
			break
		}
		at = next
		step -= sign(step)
	}
	s.Cursor = at
	return s.Shown(height)
}

// To is the panel with a row selected, or the next that is not a heading,
// shown in a height.
func (s SideContext) To(i, height int) SideContext {
	if i < 0 || i >= len(s.Rows) {
		return s
	}
	s.Cursor = s.Landing(i, 1)
	return s.Shown(height)
}

// Scroll is the panel scrolled by rows in a height, its selection kept.
func (s SideContext) Scroll(by, height int) SideContext {
	s.Top = utils.Clamp(s.Top+by, 0, max(0, len(s.Rows)-height))
	return s
}

// Shown is the panel scrolled to show its selection, and the headings just
// above it, in a height.
func (s SideContext) Shown(height int) SideContext {
	if s.Cursor >= 0 {
		top := s.Cursor
		for top > 0 && s.Rows[top-1].Heading && s.Cursor-top+1 < height {
			top--
		}
		s.Top = min(s.Top, top)
		if s.Cursor >= s.Top+height {
			s.Top = s.Cursor - height + 1
		}
	}
	s.Top = utils.Clamp(s.Top, 0, max(0, len(s.Rows)-height))
	return s
}

// At is the row at a line of the panel, or -1 for none or a heading.
func (s SideContext) At(y int) int {
	if i := s.Top + y; i >= 0 && i < len(s.Rows) && !s.Rows[i].Heading {
		return i
	}
	return -1
}

func sign(n int) int {
	if n < 0 {
		return -1
	}
	return 1
}
