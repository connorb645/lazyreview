package app

import (
	"strings"
	"time"

	"review-desk/terminal/internal/server"
)

type diffView struct {
	loc       *time.Location
	md        *markdown
	rendered  map[string]bool
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

func newDiffView(component string, files map[string]server.File, loc *time.Location, md *markdown) diffView {
	return diffView{component: component, files: files, loc: loc, md: md, rendered: map[string]bool{}, notes: map[string]bool{}, folded: map[string]bool{}, opened: map[string]bool{}, whole: map[string]bool{}}
}

func (v diffView) rebuilt(d reviewData, filter, height int) diffView {
	was := v.place()
	v.rows = buildDiff(d, v, filter)
	v.cursor = 0
	if was != nil {
		v = v.find(*was)
	}
	return v.shown(height)
}

func (v diffView) place() *server.Lines {
	if v.cursor >= len(v.rows) {
		return nil
	}
	r := v.rows[v.cursor]
	if r.kind == "block" {
		return &server.Lines{Path: r.path, Side: r.block.Side, Line: r.block.StartLine}
	}
	side, line, ok := r.at()
	if !ok {
		return &server.Lines{Path: r.path, Side: r.kind + ":" + r.key}
	}
	return &server.Lines{Path: r.path, Side: side, Line: line}
}

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
		if b := r.block; b != nil && b.Side == at.Side && at.Line >= b.StartLine && at.Line <= b.Line {
			v.cursor = i
			return v
		}
	}
	if header >= 0 {
		v.cursor = header
	}
	return v
}

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

func (v diffView) next(back bool, holds func(diffRow) bool) int {
	for i := v.cursor + step(back); i >= 0 && i < len(v.rows); i += step(back) {
		if holds(v.rows[i]) {
			return i
		}
	}
	return -1
}

func (v diffView) first(holds func(diffRow) bool) int {
	for i, r := range v.rows {
		if holds(r) {
			return i
		}
	}
	return -1
}

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

func (v diffView) file() string {
	if v.cursor < len(v.rows) {
		return v.rows[v.cursor].path
	}
	return ""
}

func (v diffView) copied() diffView {
	v.folded, v.opened, v.whole, v.notes, v.rendered = copyBools(v.folded), copyBools(v.opened), copyBools(v.whole), copyBools(v.notes), copyBools(v.rendered)
	return v
}

func copyBools(m map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, val := range m {
		out[k] = val
	}
	return out
}

func matches(r diffRow, query string) bool {
	if query == "" {
		return false
	}
	texts := []string{stripped(r.text)}
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
