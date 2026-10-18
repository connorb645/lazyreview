package helpers

import (
	"fmt"
	"strconv"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// DiffHelper opens a component's diff at a place, loading its files from
// the server first when they are not loaded yet, and moves and folds in it.
type DiffHelper struct {
	c *HelperCommon
}

// OpenPlace is the state with a component's diff in the main panel, its
// cursor on a place, or on its first row with none; or, while its files
// load, the command that loads them.
func (self *DiffHelper) OpenPlace(s context.ContextTree, component string, place *server.Lines) (context.ContextTree, tea.Cmd) {
	if s.Review == nil || component == "" {
		return s, nil
	}
	window := self.c.Helpers.Window
	if s.Screen() != context.ScreenDiff {
		s.CameFrom = s.Screen()
	}
	var missing []string
	for _, f := range presentation.DiffFiles(*s.Review, component, nil) {
		if _, ok := s.Files[f.Path]; !ok {
			missing = append(missing, f.Path)
		}
	}
	if len(missing) > 0 {
		s.Loading = true
		s = s.Showing(context.ScreenDiff)
		return s, self.loadFiles(s.Review.ID, s.Review.Page.Build, component, missing, place)
	}
	if s.Diff.Component != component || s.Diff.Files == nil {
		s.Diff = context.NewDiffContext(component, s.Files)
		s.Diff.Width = window.MainWidth(s)
	}
	s = s.Showing(context.ScreenDiff)
	s.Diff = s.Diff.Copied()
	if place != nil {
		s.Diff.Folded[place.Path] = false
		s.Diff = withRunOpen(s.Diff, place.Path, place.Side, place.Line)
	}
	s.Diff = self.Rebuilt(s, s.Diff)
	if place != nil {
		s.Diff = s.Diff.Find(*place).Placed(window.MainHeight(s))
	} else {
		s.Diff = s.Diff.To(0, window.MainHeight(s))
	}
	return s, nil
}

// AfterFiles is the state once a component's files have loaded, or failed
// to: the diff open at the place it was asked for.
func (self *DiffHelper) AfterFiles(s context.ContextTree, msg types.FilesLoaded) (context.ContextTree, tea.Cmd) {
	h := self.c.Helpers
	s.Loading = false
	if msg.Err != nil {
		s = s.Showing(s.CameFrom)
		return h.Refresh.Relaid(s.Said(msg.Err.Error(), true)), nil
	}
	files := map[string]server.File{}
	for k, v := range s.Files {
		files[k] = v
	}
	for k, v := range msg.Files {
		files[k] = v
	}
	s.Files = files
	s.Diff.Files = files
	if s.Focus() != context.MainPanel {
		return h.Panels.Followed(s)
	}
	next, cmd := self.OpenPlace(s, msg.Component, msg.Place)
	return h.Refresh.Mirrored(next), cmd
}

// ComponentOf is the component a file of the review is in.
func (self *DiffHelper) ComponentOf(s context.ContextTree, path string) string {
	return presentation.FileEntry(*s.Review, path).Component
}

// Fold is the state with the fold at the diff's cursor toggled, opened or
// closed: a note, a run of unchanged lines, or a file.
func (self *DiffHelper) Fold(s context.ContextTree, action string) context.ContextTree {
	h := self.c.Helpers.Window
	v := s.Diff.Copied()
	if v.Cursor >= len(v.Rows) {
		return s
	}
	r := v.Rows[v.Cursor]
	if r.Kind == "notehead" || r.Kind == "notetext" {
		n, _ := presentation.NoteOf(*s.Review, r.Path, r.Key)
		v.Notes[r.Key] = action == "open fold" || action == "toggle fold" && !v.NoteOpen(n)
		s.Diff = self.Rebuilt(s, v)
		if i := s.Diff.First(func(row presentation.DiffRow) bool { return row.Kind == "notehead" && row.Key == r.Key }); i >= 0 {
			s.Diff = s.Diff.To(i, h.MainHeight(s))
		}
		return s
	}
	run := r.Key != "" && r.Kind != "hunk"
	switch {
	case run && r.Kind == "gap":
		v.Opened[r.Key] = action != "close fold"
	case run:
		v.Opened[r.Key] = action == "open fold"
	case r.Kind == "file" && action == "toggle fold":
		v.Folded[r.Path] = !v.Folded[r.Path]
	default:
		v.Folded[r.Path] = action != "open fold" && !(action == "toggle fold" && v.Folded[r.Path])
	}
	s.Diff = self.Rebuilt(s, v)
	switch {
	case r.Kind != "file" && v.Folded[r.Path]:
		s.Diff = s.Diff.Find(server.Lines{Path: r.Path, Side: "file:"}).Shown(h.MainHeight(s))
	case run:
		if i := s.Diff.First(func(row presentation.DiffRow) bool { return row.Key == r.Key && row.Kind != "hunk" }); i >= 0 {
			s.Diff = s.Diff.To(i, h.MainHeight(s))
		}
	}
	return s
}

// ToLine is the state with the diff's cursor on a line of the file it is
// in, its fold and run opened for it, or a notice that the file shows no
// such line.
func (self *DiffHelper) ToLine(s context.ContextTree, n int) (context.ContextTree, tea.Cmd) {
	h := self.c.Helpers.Window
	path := s.Diff.File()
	if path == "" {
		return s, nil
	}
	s.Diff = s.Diff.Copied()
	s.Diff.Folded[path] = false
	s.Diff = withRunOpen(s.Diff, path, "RIGHT", n)
	s.Diff = self.Rebuilt(s, s.Diff)
	for i, r := range s.Diff.Rows {
		if side, line, ok := r.At(); ok && r.Path == path && (line == n && side == "RIGHT" || context.SameLine(r, server.Lines{Side: "RIGHT", Line: n})) {
			s.Diff.Cursor = i
			s.Diff = s.Diff.Placed(h.MainHeight(s))
			return s, nil
		}
	}
	return s.Said(utils.ResolvePlaceholderString(self.c.Tr.NoSuchLine, map[string]string{"path": path, "line": utils.Itoa(n)}), true), nil
}

// MatchNext is the state with the diff's cursor on the nth next row, or
// the nth previous, that holds what was searched for, or as far on as
// there are such rows.
func (self *DiffHelper) MatchNext(s context.ContextTree, back bool, n int) (context.ContextTree, tea.Cmd) {
	query := s.Diff.Query
	if query == "" {
		return s, nil
	}
	v, found := s.Diff, -1
	for ; n > 0; n-- {
		i := v.Next(back, func(r presentation.DiffRow) bool { return context.Matches(r, query) })
		if i < 0 {
			break
		}
		v.Cursor, found = i, i
	}
	if found < 0 {
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.NoMoreOf, map[string]string{"query": strconv.Quote(query)}), true), nil
	}
	s = self.c.Helpers.Jump.Leave(s)
	s.Diff = s.Diff.To(found, self.c.Helpers.Window.MainHeight(s))
	return s, nil
}

// Rebuilt is a diff built again from the review in the state, shown in
// the main panel's height.
func (self *DiffHelper) Rebuilt(s context.ContextTree, v context.DiffContext) context.DiffContext {
	return v.Rebuilt(self.c.Tr, self.c.Markdown, self.c.Location, *s.Review, s.ShownPulls(), self.c.Helpers.Window.MainHeight(s))
}

func withRunOpen(v context.DiffContext, path, side string, line int) context.DiffContext {
	file, ok := v.Files[path]
	if !ok || side == "LEFT" {
		return v
	}
	for _, g := range file.Gaps {
		if g.From <= line && line <= g.To {
			v.Opened[fmt.Sprintf("%s:%d", path, g.Before)] = true
		}
	}
	return v
}

func (self *DiffHelper) loadFiles(id, build, component string, paths []string, place *server.Lines) tea.Cmd {
	tr, c := self.c.Tr, self.c.Server
	return func() tea.Msg {
		files := map[string]server.File{}
		var mu sync.Mutex
		var wg sync.WaitGroup
		var failed error
		for _, path := range paths {
			wg.Add(1)
			go func(path string) {
				defer wg.Done()
				file, err := c.File(id, path, build)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					failed = fmt.Errorf("%s: %w", utils.ResolvePlaceholderString(tr.CouldNotOpen, map[string]string{"name": path}), err)
					return
				}
				files[path] = file
			}(path)
		}
		wg.Wait()
		return types.FilesLoaded{Component: component, Files: files, Place: place, Err: failed}
	}
}
