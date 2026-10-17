package gui

import (
	"fmt"
	"path"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func pullKey(pull int) string              { return fmt.Sprintf("pull:%d", pull) }
func componentKey(component string) string { return "component:" + component }
func fileKey(file string) string           { return "file:" + file }
func boxKey(box string) string             { return "box:" + box }

func dirKey(component, file string) string {
	return "dir:" + component + ":" + path.Dir(file)
}

func heading(text, right string, p palette) sideRow {
	return sideRow{heading: true, text: p.Muted.Render(text), right: p.Faint.Render(right)}
}

func reviewRows(d reviewData, filter pullFilter, p palette) []sideRow {
	s := d.state
	if s.Kind == "branch" {
		return []sideRow{{key: "branch", text: p.Ink.Render(s.Head.Ref) + p.Muted.Render(" → "+s.Base.Ref), it: item{kind: "branch"}}}
	}
	var rows []sideRow
	for _, m := range s.Members {
		number, title := p.pull(m.Color), p.Ink
		if !filter.shows(m.Pull) {
			number, title = p.Faint, p.Faint
		}
		right := ""
		if m.State != "" && m.State != "open" {
			right = p.Muted.Render(m.State)
		}
		rows = append(rows, sideRow{key: pullKey(m.Pull), text: number.Render(fmt.Sprintf("#%d", m.Pull)) + " " + title.Render(m.Title), right: right,
			lit: containsInt(filter, m.Pull), it: item{kind: "pull", pull: m.Pull, url: m.URL, title: m.Title}})
	}
	return rows
}

func fileRows(d reviewData, folded map[string]bool, filter pullFilter, p palette) []sideRow {
	var rows []sideRow
	for _, c := range d.page.Model.Components {
		files := diffFiles(d, c.ID, filter)
		if len(files) == 0 {
			continue
		}
		key := componentKey(c.ID)
		right := p.Muted.Render("unchanged")
		if !c.Context {
			adds, dels := 0, 0
			for _, f := range files {
				adds, dels = adds+f.Adds, dels+f.Dels
			}
			right = p.Add.Render(fmt.Sprintf("+%d", adds)) + " " + p.Del.Render(fmt.Sprintf("−%d", dels))
		}
		rows = append(rows, sideRow{key: key, text: foldMark(folded[key], p) + " " + p.componentColor(c.Color).Render("■") + " " + p.Bold.Render(c.Title), right: right,
			it: item{kind: "component", component: c.ID, title: c.Title}})
		if folded[key] {
			continue
		}
		var dirs []string
		inDir := map[string][]server.FileEntry{}
		for _, f := range files {
			dir := path.Dir(f.Path)
			if _, seen := inDir[dir]; !seen {
				dirs = append(dirs, dir)
			}
			inDir[dir] = append(inDir[dir], f)
		}
		for _, dir := range dirs {
			indent := "  "
			if dir != "." && len(dirs) > 1 {
				dk := dirKey(c.ID, inDir[dir][0].Path)
				first := server.Lines{Path: inDir[dir][0].Path, Side: "file:"}
				rows = append(rows, sideRow{key: dk, text: "  " + foldMark(folded[dk], p) + " ", title: dir, titleStyle: p.Muted, it: item{kind: "dir", component: c.ID, place: &first}})
				if folded[dk] {
					continue
				}
				indent = "    "
			}
			for _, f := range inDir[dir] {
				rows = append(rows, fileRow(d, f, indent, p))
			}
		}
	}
	return rows
}

func fileRow(d reviewData, f server.FileEntry, indent string, p palette) sideRow {
	letter := map[string]lipgloss.Style{"A": p.Add, "M": p.Warn, "D": p.Del, "R": p.Accent}[f.Status]
	mark := letter.Render(f.Status)
	if f.Status == "" {
		mark = p.Faint.Render("·")
	}
	var dots []string
	if d.page.Stack != nil {
		for _, pull := range f.Pulls {
			dots = append(dots, p.pull(memberColor(d.state, pull)).Render("●"))
		}
	}
	at := server.Lines{Path: f.Path, Side: "file:"}
	url := ""
	for i, m := range d.state.Members {
		if i == 0 || len(f.Pulls) > 0 && m.Pull == f.Pulls[0] {
			url = m.URL
		}
	}
	return sideRow{key: fileKey(f.Path), text: indent + mark + " " + p.Ink.Render(path.Base(f.Path)), right: strings.Join(dots, ""),
		it: item{kind: "file", component: f.Component, place: &at, url: url}}
}

func foldMark(folded bool, p palette) string {
	if folded {
		return p.Muted.Render("▸")
	}
	return p.Muted.Render("▾")
}

func noteListRows(d reviewData, filter pullFilter, p palette) []sideRow {
	var rows []sideRow
	stacked := d.page.Stack != nil
	if q := d.state.Queue; q != nil && q.Moved == nil {
		var queued []sideRow
		for _, m := range q.Members {
			if !filter.shows(m.Pull) {
				continue
			}
			for _, c := range m.Comments {
				at := server.Lines{Path: c.Path, Side: c.Side, Line: c.Line, StartLine: c.StartLine, StartSide: c.StartSide}
				if c.Shown != nil {
					at = *c.Shown
				}
				right := ""
				if c.Stale != "" {
					right = p.Warn.Render("stale ")
				}
				if stacked {
					right += p.pull(memberColor(d.state, m.Pull)).Render(fmt.Sprintf("#%d", m.Pull))
				}
				queued = append(queued, sideRow{key: fmt.Sprintf("queued:%d", c.ID), text: p.Accent.Render(utils.Itoa(c.ID)) + " " + p.Ink.Render(lineName(at)), right: right,
					it: item{kind: "comment", comment: c.ID, pull: m.Pull, url: m.URL, place: &at}})
			}
		}
		if len(queued) > 0 {
			rows = append(append(rows, heading("Queued", utils.Itoa(len(queued)), p)), queued...)
		}
	}
	var asked []server.Conversation
	for _, t := range d.state.Threads {
		if filter.showsFile(d, t.Path) {
			asked = append(asked, t)
		}
	}
	if len(asked) > 0 {
		rows = append(rows, heading("Claude", utils.Itoa(len(asked)), p))
		for _, t := range asked {
			at := server.Lines{Path: t.Path, Side: t.Side, Line: t.Line, StartLine: t.StartLine, StartSide: t.StartSide}
			right := ""
			if m := t.Messages; len(m) > 0 && m[len(m)-1].From == "you" {
				right = p.Muted.Render("waiting")
			}
			first := ""
			if len(t.Messages) > 0 {
				first = " " + p.Muted.Render(utils.PlainText(t.Messages[0].Text))
			}
			rows = append(rows, sideRow{key: fmt.Sprintf("conversation:%d", t.ID), text: p.Ink.Render(lineName(at)) + first, right: right,
				it: item{kind: "conversation", place: &at}})
		}
	}
	var threads []server.GitHubThread
	for _, t := range d.state.GitHub {
		if filter.shows(t.Pull) {
			threads = append(threads, t)
		}
	}
	if len(threads) > 0 {
		rows = append(rows, heading("On GitHub", utils.Itoa(len(threads)), p))
		for _, t := range threads {
			at := server.Lines{Path: t.Path, Side: "file:"}
			if t.At != nil && !t.File {
				at = *t.At
			}
			author, where, right := p.Ink, p.Muted, ""
			if t.Resolved || t.Outdated {
				author, where = p.Faint, p.Faint
				right = p.Faint.Render(map[bool]string{true: "resolved", false: "outdated"}[t.Resolved])
			}
			who := ""
			if len(t.Comments) > 0 {
				who = t.Comments[0].Author + " "
			}
			url := ""
			if len(t.Comments) > 0 {
				url = t.Comments[0].URL
			}
			rows = append(rows, sideRow{key: "thread:" + t.ID, text: author.Render(who) + where.Render(lineName(at)), right: right,
				it: item{kind: "thread", pull: t.Pull, url: url, place: &at}})
		}
	}
	if len(rows) == 0 && len(filter) > 0 {
		rows = append(rows, heading("Nothing on "+filter.words()+" yet", "", p))
	}
	if len(rows) == 0 {
		rows = append(rows, heading("Nothing queued or asked yet", "", p))
	}
	return rows
}

func lineName(at server.Lines) string {
	name := path.Base(at.Path)
	switch {
	case at.Side == "file:":
		return name
	case at.StartLine > 0 && at.StartLine != at.Line:
		return fmt.Sprintf("%s:%d-%d", name, at.StartLine, at.Line)
	}
	return fmt.Sprintf("%s:%d", name, at.Line)
}

func boxRows(d reviewData, v diagramView, filter pullFilter, p palette) []sideRow {
	l := v.level
	where := "Components"
	if l.Component != "" {
		where = componentTitle(d, l.Component)
	}
	top := heading(where, "", p)
	if l.Kind == "types" {
		top = heading(where+" › ", "", p)
		top.title, top.titleStyle = packageTitle(d.page, l.Component, l.Package), p.Muted
	}
	rows := []sideRow{top}
	for _, i := range v.grid.Order() {
		card := l.Cards[i]
		title, mark := p.Ink, ""
		if card.Kind == "component" {
			mark = p.componentColor(componentColour(d, card.ID)).Render("■") + " "
		}
		stateStyle, marked := map[string]lipgloss.Style{"new": p.Add, "changed": p.Warn, "removed": p.Del}[card.State]
		if !marked || dimmed(card, filter) {
			stateStyle = p.Faint
		}
		if dimmed(card, filter) {
			title = p.Faint
		}
		at := diagramPlace{level: l.Kind, component: l.Component, pkg: l.Package, box: card.ID}
		rows = append(rows, sideRow{key: boxKey(card.ID), text: mark, title: card.Title, titleStyle: title, right: stateStyle.Render(diagram.StateWord(card.State)), it: item{kind: "box", box: &at}})
	}
	return rows
}
