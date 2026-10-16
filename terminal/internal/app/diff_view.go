package app

import (
	"strings"
	"time"

	"review-desk/terminal/internal/server"
)

// diffView is the diff of one component: its files, each open or folded,
// whole or as a diff, with runs of unchanged lines opened or not, drawn
// unified or split, wrapped or cut, and the row the cursor is on.
type diffView struct {
	loc       *time.Location
	notes     map[string]bool
	selecting bool
	anchor    int
	component string
	files     map[string]server.File
	folded    map[string]bool
	opened    map[string]bool
	whole     map[string]bool
	split     bool
	wrap      bool
	rows      []diffRow
	cursor    int
	top       int
	query     string
	width     int
}

func newDiffView(component string, files map[string]server.File, loc *time.Location) diffView {
	return diffView{component: component, files: files, loc: loc, notes: map[string]bool{}, folded: map[string]bool{}, opened: map[string]bool{}, whole: map[string]bool{}}
}

// rebuilt draws the rows again, keeping the cursor on the same place where it still shows.
func (v diffView) rebuilt(d reviewData, filter, height int) diffView {
	was := v.place()
	v.rows = buildDiff(d, v, filter)
	v.cursor = 0
	if was != nil {
		v = v.find(*was)
	}
	return v.shown(height)
}

// place is where the cursor is: a line of a file, or a file's header.
func (v diffView) place() *server.Lines {
	if v.cursor >= len(v.rows) {
		return nil
	}
	r := v.rows[v.cursor]
	side, line, ok := r.at()
	if !ok {
		return &server.Lines{Path: r.path, Side: r.kind + ":" + r.key}
	}
	return &server.Lines{Path: r.path, Side: side, Line: line}
}

// find puts the cursor on a place's row, or on its file's header, or leaves it.
func (v diffView) find(at server.Lines) diffView {
	header := -1
	for i, r := range v.rows {
		if r.path != at.Path {
			continue
		}
		if header < 0 {
			header = i
		}
		side, line, ok := r.at()
		if ok && line == at.Line && (side == at.Side || sameLine(r, at)) {
			v.cursor = i
			return v
		}
		if !ok && r.kind+":"+r.key == at.Side {
			v.cursor = i
			return v
		}
	}
	if header >= 0 {
		v.cursor = header
	}
	return v
}

// sameLine is true when a row shows the line on its other side too, as an unchanged line does.
func sameLine(r diffRow, at server.Lines) bool {
	for _, l := range []*server.Row{r.line, r.left, r.right} {
		if l != nil && l.Kind == "same" && ((at.Side == "LEFT" && l.Old == at.Line) || (at.Side == "RIGHT" && l.New == at.Line)) {
			return true
		}
	}
	return false
}

func (v diffView) move(by, height int) diffView {
	if len(v.rows) == 0 {
		return v
	}
	v.cursor = clamp(v.cursor+by, 0, len(v.rows)-1)
	return v.shown(height)
}

func (v diffView) to(i, height int) diffView {
	return v.move(i-v.cursor, height)
}

func (v diffView) scroll(by, height int) diffView {
	v.top = clamp(v.top+by, 0, max(0, len(v.rows)-height))
	return v
}

func (v diffView) placed(height int) diffView {
	used := func(from, to int) int {
		n := 0
		for i := from; i <= to && i < len(v.rows); i++ {
			n += rowHeight(v.rows[i], v)
		}
		return n
	}
	v.top = v.cursor
	for v.top > 0 && used(v.top-1, v.cursor-1) < height/3 {
		v.top--
	}
	for v.top > 0 && used(v.top-1, len(v.rows)-1) <= height {
		v.top--
	}
	return v
}

// shown moves the first row shown so the cursor's row shows, counting the
// lines a wrapped row takes.
func (v diffView) shown(height int) diffView {
	if v.cursor < v.top {
		v.top = v.cursor
	}
	for v.top < v.cursor {
		used := 0
		for i := v.top; i <= v.cursor; i++ {
			used += rowHeight(v.rows[i], v)
		}
		if used <= height {
			break
		}
		v.top++
	}
	v.top = clamp(v.top, 0, max(0, len(v.rows)-1))
	return v
}

// next is the next row after the cursor, or the one before when back, that
// holds, or -1.
func (v diffView) next(back bool, holds func(diffRow) bool) int {
	for i := v.cursor + step(back); i >= 0 && i < len(v.rows); i += step(back) {
		if holds(v.rows[i]) {
			return i
		}
	}
	return -1
}

// first is the first row that holds, or -1.
func (v diffView) first(holds func(diffRow) bool) int {
	for i, r := range v.rows {
		if holds(r) {
			return i
		}
	}
	return -1
}

// rowAt is the row drawn on line y of the view, or -1.
func (v diffView) rowAt(y int) int {
	line := 0
	for i := v.top; i < len(v.rows); i++ {
		line += rowHeight(v.rows[i], v)
		if y < line {
			return i
		}
	}
	return -1
}

// file is the path of the file the cursor is in.
func (v diffView) file() string {
	if v.cursor < len(v.rows) {
		return v.rows[v.cursor].path
	}
	return ""
}

func (v diffView) copied() diffView {
	v.folded, v.opened, v.whole, v.notes = copyBools(v.folded), copyBools(v.opened), copyBools(v.whole), copyBools(v.notes)
	return v
}

func copyBools(m map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, val := range m {
		out[k] = val
	}
	return out
}

// matches says whether a row's text holds the query: case does not count
// unless the query has a capital.
func matches(r diffRow, query string) bool {
	if query == "" {
		return false
	}
	texts := []string{r.text}
	for _, l := range []*server.Row{r.line, r.left, r.right} {
		if l != nil {
			texts = append(texts, l.Text)
		}
	}
	for _, t := range texts {
		if query == strings.ToLower(query) {
			t = strings.ToLower(t)
		}
		if strings.Contains(t, query) {
			return true
		}
	}
	return false
}
