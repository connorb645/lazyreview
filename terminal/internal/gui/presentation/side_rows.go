package presentation

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// AllKey is the key of a stack's All row in the Review panel.
const AllKey = "all"

// PullKey is the key of a pull request's row.
func PullKey(pull int) string { return fmt.Sprintf("pull:%d", pull) }

// ComponentKey is the key of a component's row in the Files panel.
func ComponentKey(component string) string { return "component:" + component }

// FileKey is the key of a file's row in the Files panel.
func FileKey(file string) string { return "file:" + file }

// BoxKey is the key of a box's row in the Diagram panel.
func BoxKey(box string) string { return "box:" + box }

// DirKey is the key of a folder's row of a component in the Files panel.
func DirKey(component, file string) string {
	return "dir:" + component + ":" + path.Dir(file)
}

func heading(text, right string, th theme.Theme) SideRow {
	return SideRow{Heading: true, Text: th.Muted.Render(text), Right: th.Faint.Render(right)}
}

// ReviewRows are the Review panel's rows: on a stack All, then its pull
// requests, those the filter shows lit and the others dimmed; or the pull
// request, or the branch.
func ReviewRows(tr *i18n.TranslationSet, th theme.Theme, d types.ReviewData, filter types.PullFilter) []SideRow {
	s := d.State
	if s.Kind == "branch" {
		return []SideRow{{Key: "branch", Text: th.Text.Render(s.Head.Ref) + th.Muted.Render(" → "+s.Base.Ref), Item: Item{Kind: "branch"}}}
	}
	var rows []SideRow
	if d.Page.Stack != nil {
		all := th.Text
		if len(filter) > 0 {
			all = th.Faint
		}
		rows = append(rows, SideRow{Key: AllKey, Text: all.Render(tr.All), Lit: len(filter) == 0, Item: Item{Kind: "all"}})
	}
	for _, m := range s.Members {
		number, title := th.PullRequest(m.Color), th.Text
		if !filter.Shows(m.Pull) {
			number, title = th.Faint, th.Faint
		}
		right := ""
		if m.State != "" && m.State != "open" {
			right = th.Muted.Render(m.State)
		}
		rows = append(rows, SideRow{Key: PullKey(m.Pull), Text: number.Render(fmt.Sprintf("#%d", m.Pull)) + " " + title.Render(m.Title), Right: right,
			Lit: slices.Contains(filter, m.Pull), Item: Item{Kind: "pull", Pull: m.Pull, URL: m.URL, Title: m.Title}})
	}
	return rows
}

// FileRows are the Files panel's rows: each component with files the filter
// shows, its folders when it has more than one, and its files, as folded.
func FileRows(tr *i18n.TranslationSet, th theme.Theme, d types.ReviewData, folded map[string]bool, filter types.PullFilter) []SideRow {
	var rows []SideRow
	for _, c := range d.Page.Model.Components {
		files := DiffFiles(d, c.ID, filter)
		if len(files) == 0 {
			continue
		}
		key := ComponentKey(c.ID)
		right := th.Muted.Render(tr.Unchanged)
		if !c.Context {
			adds, dels := 0, 0
			for _, f := range files {
				adds, dels = adds+f.Adds, dels+f.Dels
			}
			right = th.Added.Render(fmt.Sprintf("+%d", adds)) + " " + th.Removed.Render(fmt.Sprintf("−%d", dels))
		}
		rows = append(rows, SideRow{Key: key, Text: foldMark(folded[key], th) + " " + th.Component(c.Color).Render("■") + " " + th.Bold.Render(c.Title), Right: right,
			Item: Item{Kind: "component", Component: c.ID, Title: c.Title}})
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
				dk := DirKey(c.ID, inDir[dir][0].Path)
				first := server.Lines{Path: inDir[dir][0].Path, Side: "file:"}
				rows = append(rows, SideRow{Key: dk, Text: "  " + foldMark(folded[dk], th) + " ", Title: dir, TitleStyle: th.Muted, Item: Item{Kind: "dir", Component: c.ID, Place: &first}})
				if folded[dk] {
					continue
				}
				indent = "    "
			}
			for _, f := range inDir[dir] {
				rows = append(rows, fileRow(d, f, indent, th))
			}
		}
	}
	return rows
}

func fileRow(d types.ReviewData, f server.FileEntry, indent string, th theme.Theme) SideRow {
	letter := map[string]lipgloss.Style{"A": th.Added, "M": th.Warning, "D": th.Removed, "R": th.Accent}[f.Status]
	mark := letter.Render(f.Status)
	if f.Status == "" {
		mark = th.Faint.Render("·")
	}
	var dots []string
	if d.Page.Stack != nil {
		for _, pull := range f.Pulls {
			dots = append(dots, th.PullRequest(MemberColor(d.State, pull)).Render("●"))
		}
	}
	at := server.Lines{Path: f.Path, Side: "file:"}
	url := ""
	for i, m := range d.State.Members {
		if i == 0 || len(f.Pulls) > 0 && m.Pull == f.Pulls[0] {
			url = m.URL
		}
	}
	return SideRow{Key: FileKey(f.Path), Text: indent + mark + " " + th.Text.Render(path.Base(f.Path)), Right: strings.Join(dots, ""),
		Item: Item{Kind: "file", Component: f.Component, Place: &at, URL: url}}
}

func foldMark(folded bool, th theme.Theme) string {
	if folded {
		return th.Muted.Render("▸")
	}
	return th.Muted.Render("▾")
}

// NoteListRows are the Notes panel's rows: the queued comments, the
// conversations with Claude and the threads on GitHub the filter shows.
func NoteListRows(tr *i18n.TranslationSet, th theme.Theme, d types.ReviewData, filter types.PullFilter) []SideRow {
	var rows []SideRow
	stacked := d.Page.Stack != nil
	if q := d.State.Queue; q != nil && q.Moved == nil {
		var queued []SideRow
		for _, m := range q.Members {
			if !filter.Shows(m.Pull) {
				continue
			}
			for _, c := range m.Comments {
				at := server.Lines{Path: c.Path, Side: c.Side, Line: c.Line, StartLine: c.StartLine, StartSide: c.StartSide}
				if c.Shown != nil {
					at = *c.Shown
				}
				right := ""
				if c.Stale != "" {
					right = th.Warning.Render(tr.Stale + " ")
				}
				if stacked {
					right += th.PullRequest(MemberColor(d.State, m.Pull)).Render(fmt.Sprintf("#%d", m.Pull))
				}
				queued = append(queued, SideRow{Key: fmt.Sprintf("queued:%d", c.ID), Text: th.Accent.Render(utils.Itoa(c.ID)) + " " + th.Text.Render(lineName(at)), Right: right,
					Item: Item{Kind: "comment", Comment: c.ID, Pull: m.Pull, URL: m.URL, Place: &at}})
			}
		}
		if len(queued) > 0 {
			rows = append(append(rows, heading(tr.QueuedHeading, utils.Itoa(len(queued)), th)), queued...)
		}
	}
	var asked []server.Conversation
	for _, t := range d.State.Threads {
		if ShowsFile(filter, d, t.Path) {
			asked = append(asked, t)
		}
	}
	if len(asked) > 0 {
		rows = append(rows, heading(tr.Claude, utils.Itoa(len(asked)), th))
		for _, t := range asked {
			at := server.Lines{Path: t.Path, Side: t.Side, Line: t.Line, StartLine: t.StartLine, StartSide: t.StartSide}
			right := ""
			if m := t.Messages; len(m) > 0 && m[len(m)-1].From == "you" {
				right = th.Muted.Render(tr.Waiting)
			}
			first := ""
			if len(t.Messages) > 0 {
				first = " " + th.Muted.Render(utils.PlainText(t.Messages[0].Text))
			}
			rows = append(rows, SideRow{Key: fmt.Sprintf("conversation:%d", t.ID), Text: th.Text.Render(lineName(at)) + first, Right: right,
				Item: Item{Kind: "conversation", Place: &at}})
		}
	}
	var threads []server.GitHubThread
	for _, t := range d.State.GitHub {
		if filter.Shows(t.Pull) {
			threads = append(threads, t)
		}
	}
	if len(threads) > 0 {
		rows = append(rows, heading(tr.OnGitHubHeading, utils.Itoa(len(threads)), th))
		for _, t := range threads {
			at := server.Lines{Path: t.Path, Side: "file:"}
			if t.At != nil && !t.File {
				at = *t.At
			}
			author, where, right := th.Text, th.Muted, ""
			if t.Resolved || t.Outdated {
				author, where = th.Faint, th.Faint
				right = th.Faint.Render(map[bool]string{true: tr.Resolved, false: tr.Outdated}[t.Resolved])
			}
			who := ""
			if len(t.Comments) > 0 {
				who = t.Comments[0].Author + " "
			}
			url := ""
			if len(t.Comments) > 0 {
				url = t.Comments[0].URL
			}
			rows = append(rows, SideRow{Key: "thread:" + t.ID, Text: author.Render(who) + where.Render(lineName(at)), Right: right,
				Item: Item{Kind: "thread", Pull: t.Pull, URL: url, Place: &at}})
		}
	}
	if len(rows) == 0 && len(filter) > 0 {
		rows = append(rows, heading(utils.ResolvePlaceholderString(tr.NothingOnYet, map[string]string{"pulls": FilterWords(tr, filter)}), "", th))
	}
	if len(rows) == 0 {
		rows = append(rows, heading(tr.NothingQueuedYet, "", th))
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

// BoxRows are the Diagram panel's rows: where the level is, then its boxes
// in reading order, each with its state.
func BoxRows(tr *i18n.TranslationSet, th theme.Theme, d types.ReviewData, l diagram.Level, grid diagram.Grid, filter types.PullFilter) []SideRow {
	where := tr.Components
	if l.Component != "" {
		where = ComponentTitle(d, l.Component)
	}
	top := heading(where, "", th)
	if l.Kind == "types" {
		top = heading(where+" › ", "", th)
		top.Title, top.TitleStyle = PackageTitle(d.Page, l.Component, l.Package), th.Muted
	}
	rows := []SideRow{top}
	for _, i := range grid.Order() {
		card := l.Cards[i]
		title, mark := th.Text, ""
		if card.Kind == "component" {
			mark = th.Component(ComponentColour(d, card.ID)).Render("■") + " "
		}
		stateStyle, marked := map[string]lipgloss.Style{"new": th.Added, "changed": th.Warning, "removed": th.Removed}[card.State]
		if !marked || Dimmed(card, filter) {
			stateStyle = th.Faint
		}
		if Dimmed(card, filter) {
			title = th.Faint
		}
		at := types.DiagramPlace{Level: l.Kind, Component: l.Component, Package: l.Package, Box: card.ID}
		rows = append(rows, SideRow{Key: BoxKey(card.ID), Text: mark, Title: card.Title, TitleStyle: title, Right: stateStyle.Render(diagram.StateWord(tr, card.State)), Item: Item{Kind: "box", Box: &at}})
	}
	return rows
}
