package gui

type sidePanel struct {
	rows   []sideRow
	cursor int
	top    int
}

func (s sidePanel) reset(rows []sideRow) sidePanel {
	next := sidePanel{rows: rows, top: s.top, cursor: -1}
	if was, ok := s.selected(); ok {
		next.cursor = next.index(was.key)
	}
	if next.cursor < 0 {
		next.cursor = next.landing(min(s.cursor, len(rows)-1), 1)
	}
	return next
}

func (s sidePanel) selected() (sideRow, bool) {
	if s.cursor < 0 || s.cursor >= len(s.rows) || s.rows[s.cursor].heading {
		return sideRow{}, false
	}
	return s.rows[s.cursor], true
}

func (s sidePanel) index(key string) int {
	for i, r := range s.rows {
		if r.key == key && !r.heading {
			return i
		}
	}
	return -1
}

func (s sidePanel) landing(from, by int) int {
	for i := max(0, from); i >= 0 && i < len(s.rows); i += by {
		if !s.rows[i].heading {
			return i
		}
	}
	for i := min(from, len(s.rows)-1); i >= 0 && i < len(s.rows); i -= by {
		if !s.rows[i].heading {
			return i
		}
	}
	return -1
}

func (s sidePanel) move(by, height int) sidePanel {
	if s.cursor < 0 {
		return s
	}
	at := s.cursor
	for step := by; step != 0; {
		next := s.landing(at+sign(step), sign(step))
		if next < 0 || next == at || (next-at)*sign(step) < 0 {
			break
		}
		at = next
		step -= sign(step)
	}
	s.cursor = at
	return s.shown(height)
}

func (s sidePanel) to(i, height int) sidePanel {
	if i < 0 || i >= len(s.rows) {
		return s
	}
	s.cursor = s.landing(i, 1)
	return s.shown(height)
}

func (s sidePanel) toKey(key string, height int) sidePanel {
	return s.to(s.index(key), height)
}

func (s sidePanel) scroll(by, height int) sidePanel {
	s.top = clamp(s.top+by, 0, max(0, len(s.rows)-height))
	return s
}

func (s sidePanel) shown(height int) sidePanel {
	if s.cursor >= 0 {
		top := s.cursor
		for top > 0 && s.rows[top-1].heading && s.cursor-top+1 < height {
			top--
		}
		s.top = min(s.top, top)
		if s.cursor >= s.top+height {
			s.top = s.cursor - height + 1
		}
	}
	s.top = clamp(s.top, 0, max(0, len(s.rows)-height))
	return s
}

func (s sidePanel) at(y int) int {
	if i := s.top + y; i >= 0 && i < len(s.rows) && !s.rows[i].heading {
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
