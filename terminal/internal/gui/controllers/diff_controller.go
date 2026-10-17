package controllers

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// DiffController binds the keys of the diff in the main panel: moving
// through its lines, hunks, files and notes, folding, searching, and
// commenting on lines or asking Claude about them.
type DiffController struct {
	baseController
	c *helpers.HelperCommon
}

// NewDiffController binds the keys of the diff.
func NewDiffController(c *helpers.HelperCommon) *DiffController {
	return &DiffController{c: c}
}

// Context is the diff's.
func (self *DiffController) Context() context.ContextKey {
	return context.DiffKey
}

// Bindings are the keys of every place of a review, and the diff's own.
func (self *DiffController) Bindings() []Binding {
	tr, h := self.c.Tr, self.c.Helpers
	own := []Binding{
		{Action: "down", Keys: []string{"j", "down"}, Handler: self.moved(1), Description: tr.Down},
		{Action: "up", Keys: []string{"k", "up"}, Handler: self.moved(-1), Description: tr.Up},
		{Action: "half down", Keys: []string{"ctrl+d"}, Handler: self.paged(true, false), Description: tr.HalfPageDown},
		{Action: "half up", Keys: []string{"ctrl+u"}, Handler: self.paged(true, true), Description: tr.HalfPageUp},
		{Action: "page down", Keys: []string{"ctrl+f"}, Handler: self.paged(false, false), Description: tr.PageDown},
		{Action: "page up", Keys: []string{"ctrl+b"}, Handler: self.paged(false, true), Description: tr.PageUp},
		{Action: "top", Keys: []string{"g g"}, Handler: self.end(false), Description: tr.Top},
		{Action: "bottom", Keys: []string{"G"}, Handler: self.end(true), Description: tr.Bottom},
		{Action: "next hunk", Keys: []string{"] c"}, Handler: self.hunk(false), Description: tr.NextHunk},
		{Action: "previous hunk", Keys: []string{"[ c"}, Handler: self.hunk(true), Description: tr.PreviousHunk},
		{Action: "next file", Keys: []string{"] f"}, Handler: self.next("file", false), Description: tr.NextFile},
		{Action: "previous file", Keys: []string{"[ f"}, Handler: self.next("file", true), Description: tr.PreviousFile},
		{Action: "search", Keys: []string{"/"}, Handler: self.search(false), Description: tr.Search},
		{Action: "search back", Keys: []string{"?"}, Handler: self.search(true), Description: tr.SearchBack},
		{Action: "next match", Keys: []string{"n"}, Handler: self.match(false), Description: tr.NextMatch},
		{Action: "previous match", Keys: []string{"N"}, Handler: self.match(true), Description: tr.PreviousMatch},
		{Action: "toggle fold", Keys: []string{"z a"}, Handler: self.fold("toggle fold"), Description: tr.Fold},
		{Action: "open fold", Keys: []string{"z o"}, Handler: self.fold("open fold"), Description: tr.OpenFold},
		{Action: "close fold", Keys: []string{"z c"}, Handler: self.fold("close fold"), Description: tr.CloseFold},
		{Action: "open folds", Keys: []string{"z R"}, Handler: self.folds(true), Description: tr.OpenFolds},
		{Action: "close folds", Keys: []string{"z M"}, Handler: self.folds(false), Description: tr.CloseFolds},
		{Action: "open", Keys: []string{"enter"}, Handler: stateOnly(self.open), Description: tr.ToggleFile},
		{Action: "back", Keys: []string{"-"}, Handler: stateOnly(self.back), Description: tr.Back},
		{Action: "split", Keys: []string{"space s"}, Handler: stateOnly(self.split), Description: tr.Split},
		{Action: "wrap", Keys: []string{"space w"}, Handler: stateOnly(self.wrap), Description: tr.Wrap},
		{Action: "whole", Keys: []string{"space o"}, Handler: stateOnly(self.whole), Description: tr.WholeFile},
		{Action: "rendered", Keys: []string{"space r"}, Handler: stateOnly(self.rendered), Description: tr.Rendered},
		{Action: "show type", Keys: []string{"g d"}, Handler: uncounted(h.Diagram.ShowType), Description: tr.ShowType},
		{Action: "next note", Keys: []string{"] n"}, Handler: self.next("notehead", false), Description: tr.NextNote},
		{Action: "previous note", Keys: []string{"[ n"}, Handler: self.next("notehead", true), Description: tr.PreviousNote},
		{Action: "select", Keys: []string{"V"}, Handler: stateOnly(self.selecting), Description: tr.Select},
		{Action: "other end", Keys: []string{"o"}, Handler: stateOnly(self.otherEnd), Description: tr.OtherEnd},
		{Action: "comment", Keys: []string{"c"}, Handler: uncounted(h.Editing.Comment), Description: tr.Comment},
		{Action: "ask", Keys: []string{"a"}, Handler: uncounted(h.Editing.Ask), Description: tr.AskClaude},
		{Action: "edit", Keys: []string{"e"}, Handler: uncounted(h.Editing.EditQueued), Description: tr.Edit},
		{Action: "drop", Keys: []string{"d d"}, Handler: stateOnly(h.Editing.DropQueued), Description: tr.Drop},
	}
	return join(reviewScreenBindings(self.c), withReviewAll(own))
}

func (self *DiffController) height(s context.ContextTree) int {
	return self.c.Helpers.Window.MainHeight(s)
}

func (self *DiffController) moved(by int) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		s.Diff = s.Diff.Move(by*count.N, self.height(s))
		return s, nil
	}
}

func (self *DiffController) paged(half, up bool) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		h := self.height(s)
		by := h * count.N
		if half {
			by = max(1, h/2) * count.N
		}
		if up {
			by = -by
		}
		s.Diff = s.Diff.Scroll(by, h).Move(by, h)
		return s, nil
	}
}

func (self *DiffController) end(bottom bool) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		h, v := self.height(s), s.Diff
		s = self.c.Helpers.Jump.Leave(s)
		if count.Typed {
			return self.c.Helpers.Diff.ToLine(s, count.N)
		}
		if bottom {
			s.Diff = v.To(len(v.Rows)-1, h)
		} else {
			s.Diff = v.To(0, h)
		}
		return s, nil
	}
}

func (self *DiffController) hunk(back bool) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		h, v := self.height(s), s.Diff
		isHunk := func(r presentation.DiffRow) bool { return r.Kind == "hunk" }
		for n := count.N; n > 0; n-- {
			if back {
				if i := v.Next(true, isHunk); i >= 0 {
					v = v.To(i, h)
				}
			}
			i := v.Next(back, isHunk)
			if i < 0 {
				break
			}
			v = v.To(i, h)
			if j := v.Next(false, changedRow); j >= 0 && (j == i+1 || noHunkBetween(v.Rows, i, j)) {
				v = v.To(j, h)
			}
		}
		s.Diff = v
		return s, nil
	}
}

func (self *DiffController) next(kind string, back bool) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		h, v := self.height(s), s.Diff
		for n := count.N; n > 0; n-- {
			if i := v.Next(back, func(r presentation.DiffRow) bool { return r.Kind == kind }); i >= 0 {
				v = v.To(i, h)
			}
		}
		s.Diff = v
		return s, nil
	}
}

func (self *DiffController) search(back bool) Handler {
	return func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
		return self.c.Helpers.Prompt.Search(s, back), nil
	}
}

func (self *DiffController) match(back bool) Handler {
	return func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
		return self.c.Helpers.Diff.MatchNext(s, back != s.SearchBack)
	}
}

func (self *DiffController) fold(action string) Handler {
	return func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
		return self.c.Helpers.Diff.Fold(s, action), nil
	}
}

func (self *DiffController) folds(open bool) Handler {
	return func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
		v := s.Diff.Copied()
		for _, f := range presentation.DiffFiles(*s.Review, v.Component, s.Filter) {
			v.Folded[f.Path] = !open
			for _, n := range presentation.NotesOf(*s.Review, f.Path) {
				v.Notes[n.Key] = open
			}
			for _, g := range v.Files[f.Path].Gaps {
				if open {
					v.Opened[fmt.Sprintf("%s:%d", f.Path, g.Before)] = true
				}
			}
		}
		s.Diff = self.c.Helpers.Diff.Rebuilt(s, v)
		return s, nil
	}
}

func (self *DiffController) open(s context.ContextTree) context.ContextTree {
	if v := s.Diff; v.Cursor < len(v.Rows) && (v.Rows[v.Cursor].Kind == "file" || v.Rows[v.Cursor].Kind == "gap") {
		return self.c.Helpers.Diff.Fold(s, "toggle fold")
	}
	return s
}

func (self *DiffController) back(s context.ContextTree) context.ContextTree {
	s = self.c.Helpers.Jump.Leave(s)
	s = s.Showing(s.CameFrom)
	return self.c.Helpers.Refresh.Relaid(s)
}

func (self *DiffController) split(s context.ContextTree) context.ContextTree {
	v := s.Diff
	v.Split = !v.Split
	s.Diff = self.c.Helpers.Diff.Rebuilt(s, v)
	return s
}

func (self *DiffController) wrap(s context.ContextTree) context.ContextTree {
	v := s.Diff
	v.Wrap = !v.Wrap
	s.Diff = self.c.Helpers.Diff.Rebuilt(s, v)
	return s
}

func (self *DiffController) whole(s context.ContextTree) context.ContextTree {
	v := s.Diff
	path := v.File()
	entry := presentation.FileEntry(*s.Review, path)
	if f, ok := v.Files[path]; !ok || f.Whole == nil || entry.Status == "" || len(f.Hunks) == 0 {
		return s.Said(self.c.Tr.NoWholeText, true)
	}
	v = v.Copied()
	v.Whole[path] = !v.Whole[path]
	s.Diff = self.c.Helpers.Diff.Rebuilt(s, v)
	return s
}

func (self *DiffController) rendered(s context.ContextTree) context.ContextTree {
	v := s.Diff
	path := v.File()
	if f, ok := v.Files[path]; !ok || f.Rendered == nil {
		return s.Said(self.c.Tr.NoRenderedView, true)
	}
	v = v.Copied()
	v.Rendered[path] = !v.Rendered[path]
	v.Folded[path] = false
	s.Diff = self.c.Helpers.Diff.Rebuilt(s, v)
	if i := s.Diff.First(func(r presentation.DiffRow) bool { return r.Kind == "file" && r.Path == path }); i >= 0 {
		s.Diff = s.Diff.To(i, self.height(s))
	}
	return s
}

func (self *DiffController) selecting(s context.ContextTree) context.ContextTree {
	v := s.Diff
	if len(v.Rows) == 0 {
		return s
	}
	if r := v.Rows[v.Cursor]; r.Kind != "block" {
		if _, _, ok := r.At(); !ok {
			return s.Said(self.c.Tr.SelectStartsOnLine, true)
		}
	}
	s.Diff.Selecting, s.Diff.Anchor = true, v.Cursor
	return s
}

func (self *DiffController) otherEnd(s context.ContextTree) context.ContextTree {
	if v := s.Diff; v.Selecting {
		s.Diff.Anchor, s.Diff.Cursor = v.Cursor, v.Anchor
		s.Diff = s.Diff.Shown(self.height(s))
	}
	return s
}

func changedRow(r presentation.DiffRow) bool {
	for _, l := range []*server.Row{r.Line, r.Left, r.Right} {
		if l != nil && (l.Kind == "add" || l.Kind == "del") {
			return true
		}
	}
	return false
}

func noHunkBetween(rows []presentation.DiffRow, from, to int) bool {
	for i := from + 1; i < to; i++ {
		if rows[i].Kind == "hunk" || rows[i].Kind == "file" {
			return false
		}
	}
	return true
}

// Help is the diff's keys, then those of every panel.
func (self *DiffController) Help(labels presentation.KeyLabels) []HelpSection {
	tr := self.c.Tr
	lineEnds := utils.ResolvePlaceholderString(tr.HelpLineEndsKeys, map[string]string{"top": labels.Of("top"), "bottom": labels.Of("bottom")})
	selecting := utils.ResolvePlaceholderString(tr.HelpSelect, map[string]string{"otherEnd": labels.Of("other end"), "stop": labels.Nth("close", 1)})
	dropping := utils.ResolvePlaceholderString(tr.HelpDropAtCursor, map[string]string{"confirm": labels.Of("confirm")})
	rows := []presentation.HelpRow{
		{Keys: labels.Of("down", "up"), Words: tr.HelpLinesMove},
		{Keys: labels.Of("half down", "half up", "page down", "page up"), Words: tr.HelpPages},
		{Keys: lineEnds, Words: tr.HelpLineEnds},
		{Keys: labels.Of("next hunk", "previous hunk"), Words: tr.HelpHunks},
		{Keys: labels.Of("next file", "previous file"), Words: tr.HelpFiles},
		{Keys: labels.Of("next note", "previous note"), Words: tr.HelpNotes},
		{Keys: labels.Of("search", "search back", "next match", "previous match"), Words: tr.HelpSearch},
		{Keys: labels.Of("toggle fold", "open fold", "close fold"), Words: tr.HelpFold},
		{Keys: labels.Of("open folds", "close folds"), Words: tr.HelpFoldAll},
		{Keys: labels.Of("open"), Words: tr.HelpToggleFile},
		{Keys: labels.Of("select"), Words: selecting},
		{Keys: labels.Of("comment"), Words: tr.HelpComment},
		{Keys: labels.Of("ask"), Words: tr.HelpAsk},
		{Keys: labels.Of("edit"), Words: tr.HelpEditAtCursor},
		{Keys: labels.Of("drop"), Words: dropping},
		{Keys: labels.Of("show type"), Words: tr.HelpShowType},
		{Keys: labels.Of("back"), Words: tr.HelpBack},
		{Keys: labels.Of("split"), Words: tr.HelpSplit},
		{Keys: labels.Of("wrap"), Words: tr.HelpWrap},
		{Keys: labels.Of("whole"), Words: tr.HelpWhole},
		{Keys: labels.Of("rendered"), Words: tr.HelpRendered},
	}
	return reviewScreenHelp(self.c, tr.DiffTitle, rows, labels)
}

// Hints are what the bar shows of the diff's keys: a selection's, a note's
// by its kind, or the diff's.
func (self *DiffController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	return reviewScreenHints(self.c, s, labels, func() [][2]string {
		tr := self.c.Tr
		esc := labels.Nth("close", 1)
		if s.Diff.Selecting {
			return [][2]string{{labels.Of("comment"), tr.Comment}, {labels.Of("ask"), tr.AskClaude}, {labels.Of("other end"), tr.OtherEnd}, {esc, tr.Stop}}
		}
		return append(self.noteHints(s, labels), [2]string{esc, tr.List})
	})
}

func (self *DiffController) noteHints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	tr := self.c.Tr
	fold, notes, back := [2]string{labels.Of("toggle fold"), tr.Fold}, [2]string{labels.Of("next note", "previous note"), tr.Notes}, [2]string{labels.Of("back"), tr.Back}
	if n, r, ok := self.c.Helpers.Editing.NoteAt(s); ok {
		switch {
		case n.Kind == "queued":
			return [][2]string{{labels.Of("edit"), tr.Edit}, {labels.Of("drop"), tr.Drop}, fold, notes, back}
		case n.Kind == "conversation" && r.Part == "claude":
			return [][2]string{{labels.Of("comment"), tr.CommentWithAnswer}, {labels.Of("ask"), tr.FollowUp}, fold, notes}
		case n.Kind == "conversation":
			return [][2]string{{labels.Of("ask"), tr.FollowUp}, fold, notes, back}
		}
		return [][2]string{fold, {labels.Of("github"), tr.OnGitHub}, notes, back}
	}
	return [][2]string{{labels.Of("down", "up"), tr.Move}, {labels.Of("next hunk", "previous hunk"), tr.Hunks}, {labels.Of("select"), tr.Select}, {labels.Of("comment"), tr.Comment}, {labels.Of("ask"), tr.AskClaude}, {labels.Of("search"), tr.Search}, back}
}
