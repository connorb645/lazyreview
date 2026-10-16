package app

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"review-desk/terminal/internal/server"
)

func (a App) openPlace(component string, place *server.Lines) (tea.Model, tea.Cmd) {
	if a.review == nil || component == "" {
		return a, nil
	}
	if a.screen != screenDiff {
		a.cameFrom = a.screen
	}
	var missing []string
	for _, f := range diffFiles(*a.review, component, 0) {
		if _, ok := a.files[f.Path]; !ok {
			missing = append(missing, f.Path)
		}
	}
	if len(missing) > 0 {
		a.loading = true
		a.screen = screenDiff
		return a, loadFiles(a.opts.Server, a.review.id, a.review.page.Build, component, missing, place)
	}
	if a.diff.component != component || a.diff.files == nil {
		a.diff = newDiffView(component, a.files, a.opts.Location, a.md)
		a.diff.width = a.width
	}
	a.screen = screenDiff
	a.diff = a.diff.copied()
	if place != nil {
		a.diff.folded[place.Path] = false
		a.diff = a.withRunOpen(place.Path, place.Side, place.Line)
	}
	a.diff = a.diff.rebuilt(*a.review, a.filter, a.bodyHeight())
	if place != nil {
		a.diff = a.diff.find(*place).placed(a.bodyHeight())
	} else {
		a.diff = a.diff.to(0, a.bodyHeight())
	}
	return a, nil
}

func (a App) withRunOpen(path, side string, line int) diffView {
	v := a.diff
	file, ok := v.files[path]
	if !ok || side == "LEFT" {
		return v
	}
	for _, g := range file.Gaps {
		if g.From <= line && line <= g.To {
			v.opened[fmt.Sprintf("%s:%d", path, g.Before)] = true
		}
	}
	return v
}

func (a App) componentOf(path string) string {
	return fileEntry(*a.review, path).Component
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
			a.diff = v.to(len(v.rows)-1, h)
		}
	case "next hunk", "previous hunk":
		for ; count > 0; count-- {
			back := action == "previous hunk"
			if back {
				if i := v.next(true, func(r diffRow) bool { return r.kind == "hunk" }); i >= 0 {
					v = v.to(i, h)
				}
			}
			i := v.next(back, func(r diffRow) bool { return r.kind == "hunk" })
			if i < 0 {
				break
			}
			v = v.to(i, h)
			if j := v.next(false, changedRow); j >= 0 && (j == i+1 || noHunkBetween(v.rows, i, j)) {
				v = v.to(j, h)
			}
		}
		a.diff = v
	case "next note", "previous note":
		for ; count > 0; count-- {
			if i := v.next(action == "previous note", func(r diffRow) bool { return r.kind == "notehead" }); i >= 0 {
				v = v.to(i, h)
			}
		}
		a.diff = v
	case "next file", "previous file":
		for ; count > 0; count-- {
			if i := v.next(action == "previous file", func(r diffRow) bool { return r.kind == "file" }); i >= 0 {
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
		for _, f := range diffFiles(*a.review, v.component, a.filter) {
			v.folded[f.Path] = action == "close folds"
			for _, n := range notesOf(*a.review, f.Path) {
				v.notes[n.key] = action == "open folds"
			}
			for _, g := range v.files[f.Path].Gaps {
				if action == "open folds" {
					v.opened[fmt.Sprintf("%s:%d", f.Path, g.Before)] = true
				}
			}
		}
		a.diff = v.rebuilt(*a.review, a.filter, h)
	case "open":
		if v.cursor < len(v.rows) && (v.rows[v.cursor].kind == "file" || v.rows[v.cursor].kind == "gap") {
			return a.fold("toggle fold"), nil
		}
	case "back":
		a = a.leave()
		a.screen = a.cameFrom
		return a.relaid(), nil
	case "split", "wrap":
		if action == "split" {
			v.split = !v.split
		} else {
			v.wrap = !v.wrap
		}
		a.diff = v.rebuilt(*a.review, a.filter, h)
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
		if len(v.rows) == 0 {
			return a, nil
		}
		if r := v.rows[v.cursor]; r.kind != "block" {
			if _, _, ok := r.at(); !ok {
				return a.say("V starts a selection on a line", true), nil
			}
		}
		a.diff.selecting, a.diff.anchor = true, v.cursor
	case "other end":
		if v.selecting {
			a.diff.anchor, a.diff.cursor = v.cursor, v.anchor
			a.diff = a.diff.shown(h)
		}
	case "rendered":
		path := v.file()
		if f, ok := v.files[path]; !ok || f.Rendered == nil {
			return a.say("Space r shows a Markdown file rendered, and this file has no rendered view", true), nil
		}
		v = v.copied()
		v.rendered[path] = !v.rendered[path]
		v.folded[path] = false
		a.diff = v.rebuilt(*a.review, a.filter, h)
		if i := a.diff.first(func(r diffRow) bool { return r.kind == "file" && r.path == path }); i >= 0 {
			a.diff = a.diff.to(i, h)
		}
	case "whole":
		path := v.file()
		entry := fileEntry(*a.review, path)
		if f, ok := v.files[path]; !ok || f.Whole == nil || entry.Status == "" || len(f.Hunks) == 0 {
			return a.say("this file has no whole text to show", true), nil
		}
		v = v.copied()
		v.whole[path] = !v.whole[path]
		a.diff = v.rebuilt(*a.review, a.filter, h)
	}
	return a, nil
}

func changedRow(r diffRow) bool {
	for _, l := range []*server.Row{r.line, r.left, r.right} {
		if l != nil && (l.Kind == "add" || l.Kind == "del") {
			return true
		}
	}
	return false
}

func noHunkBetween(rows []diffRow, from, to int) bool {
	for i := from + 1; i < to; i++ {
		if rows[i].kind == "hunk" || rows[i].kind == "file" {
			return false
		}
	}
	return true
}

func (a App) fold(action string) App {
	v := a.diff.copied()
	if v.cursor >= len(v.rows) {
		return a
	}
	r := v.rows[v.cursor]
	if r.kind == "notehead" || r.kind == "notetext" {
		n, _ := noteOf(*a.review, r.path, r.key)
		v.notes[r.key] = action == "open fold" || action == "toggle fold" && !v.noteOpen(n)
		a.diff = v.rebuilt(*a.review, a.filter, a.bodyHeight())
		if i := a.diff.first(func(row diffRow) bool { return row.kind == "notehead" && row.key == r.key }); i >= 0 {
			a.diff = a.diff.to(i, a.bodyHeight())
		}
		return a
	}
	run := r.key != "" && r.kind != "hunk"
	switch {
	case run && r.kind == "gap":
		v.opened[r.key] = action != "close fold"
	case run:
		v.opened[r.key] = action == "open fold"
	case r.kind == "file" && action == "toggle fold":
		v.folded[r.path] = !v.folded[r.path]
	default:
		v.folded[r.path] = action != "open fold" && !(action == "toggle fold" && v.folded[r.path])
	}
	a.diff = v.rebuilt(*a.review, a.filter, a.bodyHeight())
	switch {
	case r.kind != "file" && v.folded[r.path]:
		a.diff = a.diff.find(server.Lines{Path: r.path, Side: "file:"}).shown(a.bodyHeight())
	case run:
		if i := a.diff.first(func(row diffRow) bool { return row.key == r.key && row.kind != "hunk" }); i >= 0 {
			a.diff = a.diff.to(i, a.bodyHeight())
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
	a.diff.folded[path] = false
	a.diff = a.withRunOpen(path, "RIGHT", n)
	a.diff = a.diff.rebuilt(*a.review, a.filter, a.bodyHeight())
	for i, r := range a.diff.rows {
		if side, line, ok := r.at(); ok && r.path == path && (line == n && side == "RIGHT" || sameLine(r, server.Lines{Side: "RIGHT", Line: n})) {
			a.diff.cursor = i
			a.diff = a.diff.placed(a.bodyHeight())
			return a, nil
		}
	}
	return a.say(fmt.Sprintf("%s shows no line %d", path, n), true), nil
}

func (a App) matchNext(back bool) (tea.Model, tea.Cmd) {
	query := a.diff.query
	if query == "" {
		return a, nil
	}
	if i := a.diff.next(back, func(r diffRow) bool { return matches(r, query) }); i >= 0 {
		a = a.leave()
		a.diff = a.diff.to(i, a.bodyHeight())
		return a, nil
	}
	return a.say(fmt.Sprintf("no more of %q", query), true), nil
}

func (a App) filterBy(by int) App {
	if a.review == nil || a.review.page.Stack == nil {
		return a.say("only a stack has pull requests to filter by", true)
	}
	order := []int{0}
	for _, m := range a.review.page.Stack.Members {
		order = append(order, m.Pull)
	}
	at := 0
	for i, pull := range order {
		if pull == a.filter {
			at = i
		}
	}
	return a.setFilter(order[((at+by)%len(order)+len(order))%len(order)])
}

func (a App) setFilter(pull int) App {
	a.filter = pull
	if a.screen == screenDiff {
		a.diff = a.diff.rebuilt(*a.review, a.filter, a.bodyHeight())
	}
	return a
}

func (a App) pullCommand(arg string) App {
	if a.review == nil || a.review.page.Stack == nil {
		return a.say("only a stack has pull requests to filter by", true)
	}
	if arg == "all" {
		return a.setFilter(0)
	}
	n, err := strconv.Atoi(strings.TrimPrefix(arg, "#"))
	if err != nil || stackMember(a.review.page.Stack, n).Head == "" {
		return a.say("no pull request #"+arg+" in this stack", true)
	}
	return a.setFilter(n)
}

func (a App) here() jump {
	j := jump{screen: a.screen, item: a.rv.view.cursor, component: a.diff.component}
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
	switch j.screen {
	case screenDiff:
		a.screen = screenDiff
		return a.openPlace(j.component, j.place)
	case screenReview:
		a.screen = screenReview
		a.rv.view = a.rv.view.to(j.item, a.bodyHeight())
		return a.relaid(), nil
	case screenDiagram:
		return a.showBox(*j.box), nil
	}
	a.screen = j.screen
	return a.relaid(), nil
}

func loadFiles(c *server.Client, id, build, component string, paths []string, place *server.Lines) tea.Cmd {
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
					failed = fmt.Errorf("could not open %s: %w", path, err)
					return
				}
				files[path] = file
			}(path)
		}
		wg.Wait()
		return filesLoaded{component: component, files: files, place: place, err: failed}
	}
}
