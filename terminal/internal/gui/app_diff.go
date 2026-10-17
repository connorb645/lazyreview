package gui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func (a App) openPlace(component string, place *server.Lines) (tea.Model, tea.Cmd) {
	if a.Review == nil || component == "" {
		return a, nil
	}
	if a.Screen() != context.ScreenDiff {
		a.CameFrom = a.Screen()
	}
	var missing []string
	for _, f := range presentation.DiffFiles(*a.Review, component, nil) {
		if _, ok := a.Files[f.Path]; !ok {
			missing = append(missing, f.Path)
		}
	}
	if len(missing) > 0 {
		a.Loading = true
		a.ContextTree = a.Showing(context.ScreenDiff)
		return a, loadFiles(a.c.Tr, a.opts.Server, a.Review.ID, a.Review.Page.Build, component, missing, place)
	}
	if a.Diff.Component != component || a.Diff.Files == nil {
		a.Diff = context.NewDiffContext(component, a.Files)
		a.Diff.Width = a.mainWidth()
	}
	a.ContextTree = a.Showing(context.ScreenDiff)
	a.Diff = a.Diff.Copied()
	if place != nil {
		a.Diff.Folded[place.Path] = false
		a.Diff = a.withRunOpen(place.Path, place.Side, place.Line)
	}
	a.Diff = a.Diff.Rebuilt(a.c.Tr, a.md, a.opts.Location, *a.Review, a.Filter, a.mainHeight())
	if place != nil {
		a.Diff = a.Diff.Find(*place).Placed(a.mainHeight())
	} else {
		a.Diff = a.Diff.To(0, a.mainHeight())
	}
	return a, nil
}

func (a App) withRunOpen(path, side string, line int) context.DiffContext {
	v := a.Diff
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

func (a App) componentOf(path string) string {
	return presentation.FileEntry(*a.Review, path).Component
}

func (a App) diffAction(action string, count int, counted bool, h int) (tea.Model, tea.Cmd) {
	v := a.Diff
	switch action {
	case "down":
		a.Diff = v.Move(count, h)
	case "up":
		a.Diff = v.Move(-count, h)
	case "half down", "half up", "page down", "page up":
		by := h * count
		if strings.HasPrefix(action, "half") {
			by = max(1, h/2) * count
		}
		if strings.HasSuffix(action, "up") {
			by = -by
		}
		a.Diff = v.Scroll(by, h).Move(by, h)
	case "top", "bottom":
		a = a.leave()
		if counted {
			return a.toLine(count)
		}
		if action == "top" {
			a.Diff = v.To(0, h)
		} else {
			a.Diff = v.To(len(v.Rows)-1, h)
		}
	case "next hunk", "previous hunk":
		for ; count > 0; count-- {
			back := action == "previous hunk"
			if back {
				if i := v.Next(true, func(r presentation.DiffRow) bool { return r.Kind == "hunk" }); i >= 0 {
					v = v.To(i, h)
				}
			}
			i := v.Next(back, func(r presentation.DiffRow) bool { return r.Kind == "hunk" })
			if i < 0 {
				break
			}
			v = v.To(i, h)
			if j := v.Next(false, changedRow); j >= 0 && (j == i+1 || noHunkBetween(v.Rows, i, j)) {
				v = v.To(j, h)
			}
		}
		a.Diff = v
	case "next note", "previous note":
		for ; count > 0; count-- {
			if i := v.Next(action == "previous note", func(r presentation.DiffRow) bool { return r.Kind == "notehead" }); i >= 0 {
				v = v.To(i, h)
			}
		}
		a.Diff = v
	case "next file", "previous file":
		for ; count > 0; count-- {
			if i := v.Next(action == "previous file", func(r presentation.DiffRow) bool { return r.Kind == "file" }); i >= 0 {
				v = v.To(i, h)
			}
		}
		a.Diff = v
	case "search", "search back":
		a.SearchBack = action == "search back"
		return a.ask(map[bool]string{false: "/", true: "?"}[a.SearchBack]), nil
	case "next match", "previous match":
		return a.matchNext((action == "previous match") != a.SearchBack)
	case "toggle fold", "open fold", "close fold":
		return a.fold(action), nil
	case "open folds", "close folds":
		v = v.Copied()
		for _, f := range presentation.DiffFiles(*a.Review, v.Component, a.Filter) {
			v.Folded[f.Path] = action == "close folds"
			for _, n := range presentation.NotesOf(*a.Review, f.Path) {
				v.Notes[n.Key] = action == "open folds"
			}
			for _, g := range v.Files[f.Path].Gaps {
				if action == "open folds" {
					v.Opened[fmt.Sprintf("%s:%d", f.Path, g.Before)] = true
				}
			}
		}
		a.Diff = v.Rebuilt(a.c.Tr, a.md, a.opts.Location, *a.Review, a.Filter, h)
	case "open":
		if v.Cursor < len(v.Rows) && (v.Rows[v.Cursor].Kind == "file" || v.Rows[v.Cursor].Kind == "gap") {
			return a.fold("toggle fold"), nil
		}
	case "back":
		a = a.leave()
		a.ContextTree = a.Showing(a.CameFrom)
		return a.relaid(), nil
	case "split", "wrap":
		if action == "split" {
			v.Split = !v.Split
		} else {
			v.Wrap = !v.Wrap
		}
		a.Diff = v.Rebuilt(a.c.Tr, a.md, a.opts.Location, *a.Review, a.Filter, h)
	case "show type":
		return a.showType()
	case "comment":
		return a.comment()
	case "ask":
		return a.askClaude()
	case "edit":
		return a.editQueued()
	case "drop":
		return a.dropQueued(), nil
	case "select":
		if len(v.Rows) == 0 {
			return a, nil
		}
		if r := v.Rows[v.Cursor]; r.Kind != "block" {
			if _, _, ok := r.At(); !ok {
				return a.say(a.c.Tr.SelectStartsOnLine, true), nil
			}
		}
		a.Diff.Selecting, a.Diff.Anchor = true, v.Cursor
	case "other end":
		if v.Selecting {
			a.Diff.Anchor, a.Diff.Cursor = v.Cursor, v.Anchor
			a.Diff = a.Diff.Shown(h)
		}
	case "rendered":
		path := v.File()
		if f, ok := v.Files[path]; !ok || f.Rendered == nil {
			return a.say(a.c.Tr.NoRenderedView, true), nil
		}
		v = v.Copied()
		v.Rendered[path] = !v.Rendered[path]
		v.Folded[path] = false
		a.Diff = v.Rebuilt(a.c.Tr, a.md, a.opts.Location, *a.Review, a.Filter, h)
		if i := a.Diff.First(func(r presentation.DiffRow) bool { return r.Kind == "file" && r.Path == path }); i >= 0 {
			a.Diff = a.Diff.To(i, h)
		}
	case "whole":
		path := v.File()
		entry := presentation.FileEntry(*a.Review, path)
		if f, ok := v.Files[path]; !ok || f.Whole == nil || entry.Status == "" || len(f.Hunks) == 0 {
			return a.say(a.c.Tr.NoWholeText, true), nil
		}
		v = v.Copied()
		v.Whole[path] = !v.Whole[path]
		a.Diff = v.Rebuilt(a.c.Tr, a.md, a.opts.Location, *a.Review, a.Filter, h)
	}
	return a, nil
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

func (a App) fold(action string) App {
	v := a.Diff.Copied()
	if v.Cursor >= len(v.Rows) {
		return a
	}
	r := v.Rows[v.Cursor]
	if r.Kind == "notehead" || r.Kind == "notetext" {
		n, _ := presentation.NoteOf(*a.Review, r.Path, r.Key)
		v.Notes[r.Key] = action == "open fold" || action == "toggle fold" && !v.NoteOpen(n)
		a.Diff = v.Rebuilt(a.c.Tr, a.md, a.opts.Location, *a.Review, a.Filter, a.mainHeight())
		if i := a.Diff.First(func(row presentation.DiffRow) bool { return row.Kind == "notehead" && row.Key == r.Key }); i >= 0 {
			a.Diff = a.Diff.To(i, a.mainHeight())
		}
		return a
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
	a.Diff = v.Rebuilt(a.c.Tr, a.md, a.opts.Location, *a.Review, a.Filter, a.mainHeight())
	switch {
	case r.Kind != "file" && v.Folded[r.Path]:
		a.Diff = a.Diff.Find(server.Lines{Path: r.Path, Side: "file:"}).Shown(a.mainHeight())
	case run:
		if i := a.Diff.First(func(row presentation.DiffRow) bool { return row.Key == r.Key && row.Kind != "hunk" }); i >= 0 {
			a.Diff = a.Diff.To(i, a.mainHeight())
		}
	}
	return a
}

func (a App) toLine(n int) (tea.Model, tea.Cmd) {
	path := a.Diff.File()
	if path == "" {
		return a, nil
	}
	a.Diff = a.Diff.Copied()
	a.Diff.Folded[path] = false
	a.Diff = a.withRunOpen(path, "RIGHT", n)
	a.Diff = a.Diff.Rebuilt(a.c.Tr, a.md, a.opts.Location, *a.Review, a.Filter, a.mainHeight())
	for i, r := range a.Diff.Rows {
		if side, line, ok := r.At(); ok && r.Path == path && (line == n && side == "RIGHT" || context.SameLine(r, server.Lines{Side: "RIGHT", Line: n})) {
			a.Diff.Cursor = i
			a.Diff = a.Diff.Placed(a.mainHeight())
			return a, nil
		}
	}
	return a.say(utils.ResolvePlaceholderString(a.c.Tr.NoSuchLine, map[string]string{"path": path, "line": utils.Itoa(n)}), true), nil
}

func (a App) matchNext(back bool) (tea.Model, tea.Cmd) {
	query := a.Diff.Query
	if query == "" {
		return a, nil
	}
	if i := a.Diff.Next(back, func(r presentation.DiffRow) bool { return context.Matches(r, query) }); i >= 0 {
		a = a.leave()
		a.Diff = a.Diff.To(i, a.mainHeight())
		return a, nil
	}
	return a.say(utils.ResolvePlaceholderString(a.c.Tr.NoMoreOf, map[string]string{"query": strconv.Quote(query)}), true), nil
}

func (a App) here() context.Jump {
	j := context.Jump{Screen: a.Screen(), Item: a.ReviewView.View.Cursor, Component: a.Diff.Component}
	switch a.Screen() {
	case context.ScreenDiff:
		j.Place = a.Diff.Place()
	case context.ScreenDiagram:
		at := a.Diagram.Place()
		j.Box = &at
	}
	return j
}

func (a App) leave() App {
	a.Jumps = append(append([]context.Jump{}, a.Jumps[:a.JumpAt]...), a.here())
	a.JumpAt = len(a.Jumps)
	return a
}

func (a App) jumpBy(back bool) (tea.Model, tea.Cmd) {
	if back {
		if a.JumpAt == 0 {
			return a, nil
		}
		if a.JumpAt == len(a.Jumps) {
			a.Jumps = append(append([]context.Jump{}, a.Jumps...), a.here())
		}
		a.JumpAt--
	} else {
		if a.JumpAt+1 >= len(a.Jumps) {
			return a, nil
		}
		a.JumpAt++
	}
	j := a.Jumps[a.JumpAt]
	a.ContextTree = a.Focused(context.MainPanel, a.Driving())
	switch j.Screen {
	case context.ScreenDiff:
		a.ContextTree = a.Showing(context.ScreenDiff)
		if a.Driving() != context.NotesPanel {
			a.ContextTree = a.Focused(a.Focus(), context.FilesPanel)
		}
		return a.openPlace(j.Component, j.Place)
	case context.ScreenReview:
		a.ContextTree = a.Showing(context.ScreenReview).Focused(a.Focus(), context.ReviewPanel)
		a.ReviewView.View = a.ReviewView.View.To(j.Item, a.mainHeight())
		return a.relaid(), nil
	case context.ScreenDiagram:
		a.ContextTree = a.Focused(a.Focus(), context.DiagramPanel)
		return a.showBox(*j.Box), nil
	}
	a.ContextTree = a.Showing(j.Screen)
	return a.relaid(), nil
}

func loadFiles(tr *i18n.TranslationSet, c *server.Client, id, build, component string, paths []string, place *server.Lines) tea.Cmd {
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
