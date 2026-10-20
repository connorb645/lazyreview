package context

import (
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/gui/views"
	"github.com/connorb645/lazyreview/internal/utils"
)

// Scroller is a view of lines that belong to items: the item selected and
// the first line shown.
type Scroller struct {
	Lines  []presentation.Line
	Items  []presentation.Item
	Cursor int
	Top    int
}

// Reset is the view with new lines and items, the item the same one names
// still selected, shown in a height.
func (s Scroller) Reset(lines []presentation.Line, items []presentation.Item, same func(a, b presentation.Item) bool, height int) Scroller {
	next := Scroller{Lines: lines, Items: items, Top: s.Top}
	if s.Cursor < len(s.Items) {
		was := s.Items[s.Cursor]
		for i, it := range items {
			if same(was, it) {
				next.Cursor = i
				break
			}
		}
	}
	return next.Shown(height)
}

// Move is the view with its selection moved by items, shown in a height.
func (s Scroller) Move(by, height int) Scroller {
	if len(s.Items) == 0 {
		return s
	}
	s.Cursor = utils.Clamp(s.Cursor+by, 0, len(s.Items)-1)
	return s.Shown(height)
}

// To is the view with an item selected, shown in a height.
func (s Scroller) To(index, height int) Scroller {
	if len(s.Items) == 0 {
		return s
	}
	s.Cursor = utils.Clamp(index, 0, len(s.Items)-1)
	return s.Shown(height)
}

// Scroll is the view scrolled by lines in a height, its selection kept.
func (s Scroller) Scroll(by, height int) Scroller {
	s.Top = utils.Clamp(s.Top+by, 0, max(0, len(s.Lines)-height))
	return s
}

// Paged is the view scrolled by lines in a height, its selection moved as
// many lines on, to the item there or the next one past it, or to the
// first or last item when there is none.
func (s Scroller) Paged(by, height int) Scroller {
	from := 0
	for i, l := range s.Lines {
		if l.Item == s.Cursor {
			from = i
			break
		}
	}
	s = s.Scroll(by, height)
	if len(s.Items) == 0 || len(s.Lines) == 0 {
		return s
	}
	for i := utils.Clamp(from+by, 0, len(s.Lines)-1); i >= 0 && i < len(s.Lines); i += sign(by) {
		if s.Lines[i].Item >= 0 {
			s.Cursor = s.Lines[i].Item
			return s.Shown(height)
		}
	}
	s.Cursor = 0
	if by > 0 {
		s.Cursor = len(s.Items) - 1
	}
	return s.Shown(height)
}

// Shown is the view scrolled to show as much of the selected item as fits
// in a height.
func (s Scroller) Shown(height int) Scroller {
	first, last := -1, -1
	for i, l := range s.Lines {
		if l.Item == s.Cursor {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 || height <= 0 {
		s.Top = utils.Clamp(s.Top, 0, max(0, len(s.Lines)-height))
		return s
	}
	if last-first+1 > height {
		last = first + height - 1
	}
	if first < s.Top {
		s.Top = first
	}
	if last >= s.Top+height {
		s.Top = last - height + 1
	}
	s.Top = utils.Clamp(s.Top, 0, max(0, len(s.Lines)-height))
	return s
}

// Extent is how many lines the view holds and the first shown.
func (s Scroller) Extent() views.Extent {
	return views.Extent{Lines: len(s.Lines), Top: s.Top}
}

// At is the item at a line of the view, or -1.
func (s Scroller) At(y int) int {
	if i := s.Top + y; i >= 0 && i < len(s.Lines) {
		return s.Lines[i].Item
	}
	return -1
}
