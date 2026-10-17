package gui

type scroller struct {
	lines  []line
	items  []item
	cursor int
	top    int
}

func (s scroller) reset(lines []line, items []item, same func(a, b item) bool, height int) scroller {
	next := scroller{lines: lines, items: items, top: s.top}
	if s.cursor < len(s.items) {
		was := s.items[s.cursor]
		for i, it := range items {
			if same(was, it) {
				next.cursor = i
				break
			}
		}
	}
	return next.shown(height)
}

func (s scroller) move(by, height int) scroller {
	if len(s.items) == 0 {
		return s
	}
	s.cursor = clamp(s.cursor+by, 0, len(s.items)-1)
	return s.shown(height)
}

func (s scroller) to(index, height int) scroller {
	if len(s.items) == 0 {
		return s
	}
	s.cursor = clamp(index, 0, len(s.items)-1)
	return s.shown(height)
}

func (s scroller) scroll(by, height int) scroller {
	s.top = clamp(s.top+by, 0, max(0, len(s.lines)-height))
	return s
}

func (s scroller) shown(height int) scroller {
	first, last := -1, -1
	for i, l := range s.lines {
		if l.item == s.cursor {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 || height <= 0 {
		s.top = clamp(s.top, 0, max(0, len(s.lines)-height))
		return s
	}
	if last-first+1 > height {
		last = first + height - 1
	}
	if first < s.top {
		s.top = first
	}
	if last >= s.top+height {
		s.top = last - height + 1
	}
	s.top = clamp(s.top, 0, max(0, len(s.lines)-height))
	return s
}

func (s scroller) extent() extent {
	return extent{lines: len(s.lines), top: s.top}
}

func (s scroller) at(y int) int {
	if i := s.top + y; i >= 0 && i < len(s.lines) {
		return s.lines[i].item
	}
	return -1
}

func around(n, size int) int {
	return (n%size + size) % size
}

func clamp(n, low, high int) int {
	if n < low {
		return low
	}
	if n > high {
		return high
	}
	return n
}
