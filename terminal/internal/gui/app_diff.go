package gui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func (a App) openPlace(component string, place *server.Lines) (tea.Model, tea.Cmd) {
	if a.review == nil || component == "" {
		return a, nil
	}
	if a.screen != screenDiff {
		a.cameFrom = a.screen
	}
	var missing []string
	for _, f := range presentation.DiffFiles(*a.review, component, nil) {
		if _, ok := a.files[f.Path]; !ok {
			missing = append(missing, f.Path)
		}
	}
	if len(missing) > 0 {
		a.loading = true
		a.screen = screenDiff
		return a, loadFiles(a.c.Tr, a.opts.Server, a.review.ID, a.review.Page.Build, component, missing, place)
	}
	if a.diff.Component != component || a.diff.Files == nil {
		a.diff = newDiffView(component, a.files)
		a.diff.Width = a.mainWidth()
	}
	a.screen = screenDiff
	a.diff = a.diff.copied()
	if place != nil {
		a.diff.Folded[place.Path] = false
		a.diff = a.withRunOpen(place.Path, place.Side, place.Line)
	}
	a.diff = a.diff.rebuilt(a.c.Tr, a.md, a.opts.Location, *a.review, a.filter, a.mainHeight())
	if place != nil {
		a.diff = a.diff.find(*place).placed(a.mainHeight())
	} else {
		a.diff = a.diff.to(0, a.mainHeight())
	}
	return a, nil
}

func (a App) withRunOpen(path, side string, line int) diffView {
	v := a.diff
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
	return presentation.FileEntry(*a.review, path).Component
}

func (a App) diffAction(action string, count int, counted bool, h int) (tea.Model, tea.Cmd) {
	v := a.diff
	switch action {
	case "down":
		a.diff = v.move(count, h)
	case "up":
		a.diff = v.move(-count, h)
	case "half down", "half up", "page down", "page up":
		by := h * count
		if strings.HasPrefix(action, "half") {
			by = max(1, h/2) * count
		}
		if strings.HasSuffix(action, "up") {
			by = -by
		}
		a.diff = v.scroll(by, h).move(by, h)
	case "top", "bottom":
		a = a.leave()
		if counted {
			return a.toLine(count)
		}
		if action == "top" {
			a.diff = v.to(0, h)
		} else {
			a.diff = v.to(len(v.Rows)-1, h)
		}
	case "next hunk", "previous hunk":
		for ; count > 0; count-- {
			back := action == "previous hunk"
			if back {
				if i := v.next(true, func(r presentation.DiffRow) bool { return r.Kind == "hunk" }); i >= 0 {
					v = v.to(i, h)
				}
			}
			i := v.next(back, func(r presentation.DiffRow) bool { return r.Kind == "hunk" })
			if i < 0 {
				break
			}
			v = v.to(i, h)
			if j := v.next(false, changedRow); j >= 0 && (j == i+1 || noHunkBetween(v.Rows, i, j)) {
				v = v.to(j, h)
			}
		}
		a.diff = v
	case "next note", "previous note":
		for ; count > 0; count-- {
			if i := v.next(action == "previous note", func(r presentation.DiffRow) bool { return r.Kind == "notehead" }); i >= 0 {
				v = v.to(i, h)
			}
		}
		a.diff = v
	case "next file", "previous file":
		for ; count > 0; count-- {
			if i := v.next(action == "previous file", func(r presentation.DiffRow) bool { return r.Kind == "file" }); i >= 0 {
				v = v.to(i, h)
			}
		}
		a.diff = v
	case "search", "search back":
		a.searchBack = action == "search back"
		return a.ask(map[bool]string{false: "/", true: "?"}[a.searchBack]), nil
	case "next match", "previous match":
		return a.matchNext((action == "previous match") != a.searchBack)
	case "toggle fold", "open fold", "close fold":
		return a.fold(action), nil
	case "open folds", "close folds":
		v = v.copied()
		for _, f := range presentation.DiffFiles(*a.review, v.Component, a.filter) {
			v.Folded[f.Path] = action == "close folds"
			for _, n := range presentation.NotesOf(*a.review, f.Path) {
				v.Notes[n.Key] = action == "open folds"
			}
			for _, g := range v.Files[f.Path].Gaps {
				if action == "open folds" {
					v.Opened[fmt.Sprintf("%s:%d", f.Path, g.Before)] = true
				}
			}
		}
		a.diff = v.rebuilt(a.c.Tr, a.md, a.opts.Location, *a.review, a.filter, h)
	case "open":
		if v.Cursor < len(v.Rows) && (v.Rows[v.Cursor].Kind == "file" || v.Rows[v.Cursor].Kind == "gap") {
			return a.fold("toggle fold"), nil
		}
	case "back":
		a = a.leave()
		a.screen = a.cameFrom
		return a.relaid(), nil
	case "split", "wrap":
		if action == "split" {
			v.Split = !v.Split
		} else {
			v.Wrap = !v.Wrap
		}
		a.diff = v.rebuilt(a.c.Tr, a.md, a.opts.Location, *a.review, a.filter, h)
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
		a.diff.Selecting, a.diff.Anchor = true, v.Cursor
	case "other end":
		if v.Selecting {
			a.diff.Anchor, a.diff.Cursor = v.Cursor, v.Anchor
			a.diff = a.diff.shown(h)
		}
	case "rendered":
		path := v.file()
		if f, ok := v.Files[path]; !ok || f.Rendered == nil {
			return a.say(a.c.Tr.NoRenderedView, true), nil
		}
		v = v.copied()
		v.Rendered[path] = !v.Rendered[path]
		v.Folded[path] = false
		a.diff = v.rebuilt(a.c.Tr, a.md, a.opts.Location, *a.review, a.filter, h)
		if i := a.diff.first(func(r presentation.DiffRow) bool { return r.Kind == "file" && r.Path == path }); i >= 0 {
			a.diff = a.diff.to(i, h)
		}
	case "whole":
		path := v.file()
		entry := presentation.FileEntry(*a.review, path)
		if f, ok := v.Files[path]; !ok || f.Whole == nil || entry.Status == "" || len(f.Hunks) == 0 {
			return a.say(a.c.Tr.NoWholeText, true), nil
		}
		v = v.copied()
		v.Whole[path] = !v.Whole[path]
		a.diff = v.rebuilt(a.c.Tr, a.md, a.opts.Location, *a.review, a.filter, h)
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
	v := a.diff.copied()
	if v.Cursor >= len(v.Rows) {
		return a
	}
	r := v.Rows[v.Cursor]
	if r.Kind == "notehead" || r.Kind == "notetext" {
		n, _ := presentation.NoteOf(*a.review, r.Path, r.Key)
		v.Notes[r.Key] = action == "open fold" || action == "toggle fold" && !v.NoteOpen(n)
		a.diff = v.rebuilt(a.c.Tr, a.md, a.opts.Location, *a.review, a.filter, a.mainHeight())
		if i := a.diff.first(func(row presentation.DiffRow) bool { return row.Kind == "notehead" && row.Key == r.Key }); i >= 0 {
			a.diff = a.diff.to(i, a.mainHeight())
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
	a.diff = v.rebuilt(a.c.Tr, a.md, a.opts.Location, *a.review, a.filter, a.mainHeight())
	switch {
	case r.Kind != "file" && v.Folded[r.Path]:
		a.diff = a.diff.find(server.Lines{Path: r.Path, Side: "file:"}).shown(a.mainHeight())
	case run:
		if i := a.diff.first(func(row presentation.DiffRow) bool { return row.Key == r.Key && row.Kind != "hunk" }); i >= 0 {
			a.diff = a.diff.to(i, a.mainHeight())
		}
	}
	return a
}

func (a App) toLine(n int) (tea.Model, tea.Cmd) {
	path := a.diff.file()
	if path == "" {
		return a, nil
	}
	a.diff = a.diff.copied()
	a.diff.Folded[path] = false
	a.diff = a.withRunOpen(path, "RIGHT", n)
	a.diff = a.diff.rebuilt(a.c.Tr, a.md, a.opts.Location, *a.review, a.filter, a.mainHeight())
	for i, r := range a.diff.Rows {
		if side, line, ok := r.At(); ok && r.Path == path && (line == n && side == "RIGHT" || sameLine(r, server.Lines{Side: "RIGHT", Line: n})) {
			a.diff.Cursor = i
			a.diff = a.diff.placed(a.mainHeight())
			return a, nil
		}
	}
	return a.say(utils.ResolvePlaceholderString(a.c.Tr.NoSuchLine, map[string]string{"path": path, "line": utils.Itoa(n)}), true), nil
}

func (a App) matchNext(back bool) (tea.Model, tea.Cmd) {
	query := a.diff.query
	if query == "" {
		return a, nil
	}
	if i := a.diff.next(back, func(r presentation.DiffRow) bool { return matches(r, query) }); i >= 0 {
		a = a.leave()
		a.diff = a.diff.to(i, a.mainHeight())
		return a, nil
	}
	return a.say(utils.ResolvePlaceholderString(a.c.Tr.NoMoreOf, map[string]string{"query": strconv.Quote(query)}), true), nil
}

func (a App) here() jump {
	j := jump{screen: a.screen, item: a.rv.view.cursor, component: a.diff.Component}
	switch a.screen {
	case screenDiff:
		j.place = a.diff.place()
	case screenDiagram:
		at := a.dg.place()
		j.box = &at
	}
	return j
}

func (a App) leave() App {
	a.jumps = append(append([]jump{}, a.jumps[:a.jumpAt]...), a.here())
	a.jumpAt = len(a.jumps)
	return a
}

func (a App) jumpBy(back bool) (tea.Model, tea.Cmd) {
	if back {
		if a.jumpAt == 0 {
			return a, nil
		}
		if a.jumpAt == len(a.jumps) {
			a.jumps = append(append([]jump{}, a.jumps...), a.here())
		}
		a.jumpAt--
	} else {
		if a.jumpAt+1 >= len(a.jumps) {
			return a, nil
		}
		a.jumpAt++
	}
	j := a.jumps[a.jumpAt]
	a.focus = mainPanel
	switch j.screen {
	case screenDiff:
		a.screen = screenDiff
		if a.driving != notesPanel {
			a.driving = filesPanel
		}
		return a.openPlace(j.component, j.place)
	case screenReview:
		a.screen, a.driving = screenReview, reviewPanel
		a.rv.view = a.rv.view.to(j.item, a.mainHeight())
		return a.relaid(), nil
	case screenDiagram:
		a.driving = diagramPanel
		return a.showBox(*j.box), nil
	}
	a.screen = j.screen
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
		return filesLoaded{component: component, files: files, place: place, err: failed}
	}
}
