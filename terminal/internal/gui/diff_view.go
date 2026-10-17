package gui

import (
	"strings"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

type diffView struct {
	presentation.DiffView
	query string
}

func newDiffView(component string, files map[string]server.File) diffView {
	return diffView{DiffView: presentation.DiffView{Component: component, Files: files, Rendered: map[string]bool{}, Notes: map[string]bool{}, Folded: map[string]bool{}, Opened: map[string]bool{}, Whole: map[string]bool{}}}
}

func (v diffView) rebuilt(tr *i18n.TranslationSet, md *presentation.Markdown, loc *time.Location, d types.ReviewData, filter types.PullFilter, height int) diffView {
	was := v.place()
	v.Rows = presentation.BuildDiff(tr, md, loc, d, v.DiffView, filter)
	v.Cursor = 0
	if was != nil {
		v = v.find(*was)
	}
	return v.shown(height)
}

func (v diffView) place() *server.Lines {
	if v.Cursor >= len(v.Rows) {
		return nil
	}
	r := v.Rows[v.Cursor]
	if r.Kind == "block" {
		return &server.Lines{Path: r.Path, Side: r.Block.Side, Line: r.Block.StartLine}
	}
	side, line, ok := r.At()
	if !ok {
		return &server.Lines{Path: r.Path, Side: r.Kind + ":" + r.Key}
	}
	return &server.Lines{Path: r.Path, Side: side, Line: line}
}

func (v diffView) find(at server.Lines) diffView {
	header := -1
	for i, r := range v.Rows {
		if r.Path != at.Path {
			continue
		}
		if header < 0 {
			header = i
		}
		side, line, ok := r.At()
		if ok && line == at.Line && (side == at.Side || sameLine(r, at)) {
			v.Cursor = i
			return v
		}
		if !ok && r.Kind+":"+r.Key == at.Side {
			v.Cursor = i
			return v
		}
		if b := r.Block; b != nil && b.Side == at.Side && at.Line >= b.StartLine && at.Line <= b.Line {
			v.Cursor = i
			return v
		}
	}
	if header >= 0 {
		v.Cursor = header
	}
	return v
}

func sameLine(r presentation.DiffRow, at server.Lines) bool {
	for _, l := range []*server.Row{r.Line, r.Left, r.Right} {
		if l != nil && l.Kind == "same" && ((at.Side == "LEFT" && l.Old == at.Line) || (at.Side == "RIGHT" && l.New == at.Line)) {
			return true
		}
	}
	return false
}

func (v diffView) move(by, height int) diffView {
	if len(v.Rows) == 0 {
		return v
	}
	v.Cursor = utils.Clamp(v.Cursor+by, 0, len(v.Rows)-1)
	return v.shown(height)
}

func (v diffView) to(i, height int) diffView {
	return v.move(i-v.Cursor, height)
}

func (v diffView) scroll(by, height int) diffView {
	v.Top = utils.Clamp(v.Top+by, 0, v.lastTop(height))
	return v
}

func (v diffView) placed(height int) diffView {
	used := func(from, to int) int {
		n := 0
		for i := from; i <= to && i < len(v.Rows); i++ {
			n += presentation.RowHeight(v.Rows[i], v.DiffView)
		}
		return n
	}
	v.Top = v.Cursor
	for v.Top > 0 && used(v.Top-1, v.Cursor-1) < height/3 {
		v.Top--
	}
	v.Top = min(v.Top, v.lastTop(height))
	return v
}

func (v diffView) atTop(height int) diffView {
	v.Top = min(v.Cursor, v.lastTop(height))
	return v
}

func (v diffView) lastTop(height int) int {
	used := 0
	for i := len(v.Rows) - 1; i >= 0; i-- {
		if used += presentation.RowHeight(v.Rows[i], v.DiffView); used > height {
			return min(i+1, len(v.Rows)-1)
		}
	}
	return 0
}

func (v diffView) shown(height int) diffView {
	if v.Cursor < v.Top {
		v.Top = v.Cursor
	}
	for v.Top < v.Cursor {
		used := 0
		for i := v.Top; i <= v.Cursor; i++ {
			used += presentation.RowHeight(v.Rows[i], v.DiffView)
		}
		if used <= height {
			break
		}
		v.Top++
	}
	v.Top = utils.Clamp(v.Top, 0, max(0, len(v.Rows)-1))
	return v
}

func (v diffView) next(back bool, holds func(presentation.DiffRow) bool) int {
	for i := v.Cursor + step(back); i >= 0 && i < len(v.Rows); i += step(back) {
		if holds(v.Rows[i]) {
			return i
		}
	}
	return -1
}

func (v diffView) first(holds func(presentation.DiffRow) bool) int {
	for i, r := range v.Rows {
		if holds(r) {
			return i
		}
	}
	return -1
}

func (v diffView) extent() views.Extent {
	var at views.Extent
	for i, r := range v.Rows {
		if i == v.Top {
			at.Top = at.Lines
		}
		at.Lines += presentation.RowHeight(r, v.DiffView)
	}
	return at
}

func (v diffView) rowAt(y int) int {
	line := 0
	for i := v.Top; i < len(v.Rows); i++ {
		line += presentation.RowHeight(v.Rows[i], v.DiffView)
		if y < line {
			return i
		}
	}
	return -1
}

func (v diffView) file() string {
	if v.Cursor < len(v.Rows) {
		return v.Rows[v.Cursor].Path
	}
	return ""
}

func (v diffView) copied() diffView {
	v.Folded, v.Opened, v.Whole, v.Notes, v.Rendered = copyBools(v.Folded), copyBools(v.Opened), copyBools(v.Whole), copyBools(v.Notes), copyBools(v.Rendered)
	return v
}

func copyBools(m map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, val := range m {
		out[k] = val
	}
	return out
}

func matches(r presentation.DiffRow, query string) bool {
	if query == "" {
		return false
	}
	texts := []string{utils.Stripped(r.Text)}
	for _, l := range []*server.Row{r.Line, r.Left, r.Right} {
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
