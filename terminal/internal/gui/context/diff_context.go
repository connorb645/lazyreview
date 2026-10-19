package context

import (
	"slices"
	"strings"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// DiffContext is the diff in the main panel of the files the Files panel
// has chosen, and what was last searched for in it.
type DiffContext struct {
	presentation.DiffView
	Query string
}

// NewDiffContext is a diff of the files loaded, its lines numbered as the
// config says, with nothing folded or opened.
func NewDiffContext(files map[string]models.File, numbers presentation.LineNumbers) DiffContext {
	return DiffContext{DiffView: presentation.DiffView{Files: files, Numbers: numbers, Rendered: map[string]bool{}, Notes: map[string]bool{}, Folded: map[string]bool{}, Opened: map[string]bool{}, Whole: map[string]bool{}}}
}

// Rebuilt is the diff of files made again from the review, its cursor kept
// on its place, shown in a height. Its code goes back to the first column
// when the files are not the ones it showed.
func (v DiffContext) Rebuilt(tr *i18n.TranslationSet, md *presentation.Markdown, loc *time.Location, d types.ReviewData, files []models.FileEntry, filter types.PullFilter, height int) DiffContext {
	was, before := v.Place(), v.paths()
	v.Rows = presentation.BuildDiff(tr, md, loc, d, v.DiffView, files, filter)
	if !slices.Equal(before, v.paths()) {
		v.Left = 0
	}
	v.Cursor = 0
	if was != nil {
		v = v.Find(*was)
	}
	return v.Shown(height)
}

// Place is where the cursor is, as lines of a file or one of its parts.
func (v DiffContext) Place() *models.Lines {
	if v.Cursor >= len(v.Rows) {
		return nil
	}
	r := v.Rows[v.Cursor]
	if r.Kind == "block" {
		return &models.Lines{Path: r.Path, Side: r.Block.Side, Line: r.Block.StartLine}
	}
	side, line, ok := r.At()
	if !ok {
		return &models.Lines{Path: r.Path, Side: r.Kind + ":" + r.Key}
	}
	return &models.Lines{Path: r.Path, Side: side, Line: line}
}

// Find is the diff with the cursor on a place, or on its file's header.
func (v DiffContext) Find(at models.Lines) DiffContext {
	header := -1
	for i, r := range v.Rows {
		if r.Path != at.Path {
			continue
		}
		if header < 0 {
			header = i
		}
		side, line, ok := r.At()
		if ok && line == at.Line && (side == at.Side || SameLine(r, at)) || r.Left != nil && at.Side == "LEFT" && r.Left.Old == at.Line {
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

// SameLine says whether a row shows an unchanged line, on either side, at
// a place.
func SameLine(r presentation.DiffRow, at models.Lines) bool {
	for _, l := range []*models.Row{r.Line, r.Left, r.Right} {
		if l != nil && l.Kind == "same" && ((at.Side == "LEFT" && l.Old == at.Line) || (at.Side == "RIGHT" && l.New == at.Line)) {
			return true
		}
	}
	return false
}

// Move is the diff with the cursor moved by rows, shown in a height.
func (v DiffContext) Move(by, height int) DiffContext {
	if len(v.Rows) == 0 {
		return v
	}
	v.Cursor = utils.Clamp(v.Cursor+by, 0, len(v.Rows)-1)
	return v.Shown(height)
}

// To is the diff with the cursor on a row, shown in a height.
func (v DiffContext) To(i, height int) DiffContext {
	return v.Move(i-v.Cursor, height)
}

// Sideways is the diff with its code scrolled by columns, to the right when
// by is more than nought, from the first column to where the widest line of
// code shown in a height ends at the right edge.
func (v DiffContext) Sideways(by, height int) DiffContext {
	end := presentation.ScrollEnd(v.DiffView, height)
	switch {
	case by < 0:
		v.Left = max(0, v.Left+by)
	case v.Left < end:
		v.Left = min(end, v.Left+by)
	}
	return v
}

// Scroll is the diff scrolled by lines in a height.
func (v DiffContext) Scroll(by, height int) DiffContext {
	v.Top = utils.Clamp(v.Top+by, 0, v.lastTop(height))
	return v
}

// Placed is the diff scrolled so the cursor sits a third of a height down.
func (v DiffContext) Placed(height int) DiffContext {
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

// AtTop is the diff scrolled so the cursor is its first row shown.
func (v DiffContext) AtTop(height int) DiffContext {
	v.Top = min(v.Cursor, v.lastTop(height))
	return v
}

func (v DiffContext) lastTop(height int) int {
	used := 0
	for i := len(v.Rows) - 1; i >= 0; i-- {
		if used += presentation.RowHeight(v.Rows[i], v.DiffView); used > height {
			return min(i+1, len(v.Rows)-1)
		}
	}
	return 0
}

// Shown is the diff scrolled to show the cursor in a height.
func (v DiffContext) Shown(height int) DiffContext {
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

// Next is the next row after the cursor, or before it, that holds, or -1.
func (v DiffContext) Next(back bool, holds func(presentation.DiffRow) bool) int {
	for i := v.Cursor + step(back); i >= 0 && i < len(v.Rows); i += step(back) {
		if holds(v.Rows[i]) {
			return i
		}
	}
	return -1
}

// First is the first row that holds, or -1.
func (v DiffContext) First(holds func(presentation.DiffRow) bool) int {
	for i, r := range v.Rows {
		if holds(r) {
			return i
		}
	}
	return -1
}

// Extent is how many lines the diff takes and the first one shown.
func (v DiffContext) Extent() views.Extent {
	var at views.Extent
	for i, r := range v.Rows {
		if i == v.Top {
			at.Top = at.Lines
		}
		at.Lines += presentation.RowHeight(r, v.DiffView)
	}
	return at
}

// RowAt is the row at a line of the main panel, or -1.
func (v DiffContext) RowAt(y int) int {
	line := 0
	for i := v.Top; i < len(v.Rows); i++ {
		line += presentation.RowHeight(v.Rows[i], v.DiffView)
		if y < line {
			return i
		}
	}
	return -1
}

// File is the path of the file the cursor is in.
func (v DiffContext) File() string {
	if v.Cursor < len(v.Rows) {
		return v.Rows[v.Cursor].Path
	}
	return ""
}

// AtFileEnd is the diff with the cursor on the first line of the file it
// is in, that file's header shown above it, or on its last line, shown in a
// height. A line is a line of code or a block of a rendered file; a file
// that shows none, such as a folded one, ends at its header.
func (v DiffContext) AtFileEnd(last bool, height int) DiffContext {
	path := v.File()
	header, first, end := -1, -1, -1
	for i, r := range v.Rows {
		if r.Path != path {
			continue
		}
		if header < 0 {
			header = i
		}
		if _, _, ok := r.At(); !ok && r.Kind != "block" {
			continue
		}
		if first < 0 {
			first = i
		}
		end = i
	}
	switch {
	case header < 0:
		return v
	case first < 0:
		return v.To(header, height)
	case last:
		return v.To(end, height)
	}
	return v.To(header, height).AtTop(height).To(first, height)
}

func (v DiffContext) paths() []string {
	var out []string
	for _, r := range v.Rows {
		if r.Kind == "file" {
			out = append(out, r.Path)
		}
	}
	return out
}

// Copied is the diff with its own copies of its folds, so changing them
// changes no other copy.
func (v DiffContext) Copied() DiffContext {
	v.Folded, v.Opened, v.Whole, v.Notes, v.Rendered = utils.CopyBools(v.Folded), utils.CopyBools(v.Opened), utils.CopyBools(v.Whole), utils.CopyBools(v.Notes), utils.CopyBools(v.Rendered)
	return v
}

// Matches says whether a row holds the words searched for, case and all
// when the search has a capital.
func Matches(r presentation.DiffRow, query string) bool {
	if query == "" {
		return false
	}
	texts := []string{utils.Stripped(r.Text)}
	for _, l := range []*models.Row{r.Line, r.Left, r.Right} {
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
