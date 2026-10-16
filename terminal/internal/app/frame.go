package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// View draws the screen: the panels on the left, the main panel on the
// right, any popup over them, and a bar of the keys that matter here. The
// home screen has a panel on the left for each repository; a review has its
// four.
func (a App) View() string {
	if a.width == 0 || a.height == 0 {
		return ""
	}
	rows := make([]string, max(0, a.height-1))
	place := func(r rect, lines []string) {
		for i, line := range lines {
			if y := r.y + i; y >= 0 && y < len(rows) {
				rows[y] += line
			}
		}
	}
	if a.screen == screenHome {
		l := a.homeLayout()
		for i, r := range l.panels {
			if r.w > 2 && r.h > 0 {
				place(r, a.homePanelBox(i, r, a.home.focus == i && !a.popped()))
			}
		}
		if r := l.main; r.shown() {
			place(r, a.homeMainBox(r))
		}
	} else {
		l := a.layout()
		for n := reviewPanel; n <= sideCount; n++ {
			if r := l.sides[n]; r.w > 2 && r.h > 0 {
				place(r, a.sideBox(n, r, a.focus == n && !a.popped()))
			}
		}
		if r := l.main; r.shown() {
			place(r, a.mainBox(r, a.focus == mainPanel && !a.popped()))
		}
	}
	for i := range rows {
		rows[i] = fit(rows[i], a.width)
	}
	if box := a.popup(); len(box) > 0 {
		w := ansi.StringWidth(box[0])
		rows = overlay(rows, box, max(0, (a.width-w)/2), max(0, (len(rows)-len(box))/2), a.p)
	}
	return strings.Join(append(rows, a.barLine()), "\n")
}

func (a App) popped() bool {
	return a.menu != nil || a.posting != nil || a.confirming != nil
}

func (a App) sideBox(n int, r rect, focused bool) []string {
	content, place := sideContent(a.sides[n], r.inner(), focused, a.p)
	return framed(content, extent{}, r.w, r.h, panelTitle(n, panelNames[n]), a.tabs(n, r.inner().w), place, focused, a.p)
}

func sideContent(s sidePanel, in rect, focused bool, p palette) ([]string, string) {
	var content []string
	for i := s.top; i < s.top+in.h && i < len(s.rows); i++ {
		line := sideLine(s.rows[i], in.w)
		if i == s.cursor && !s.rows[i].heading {
			line = p.highlight(line, in.w, focused)
		}
		content = append(content, line)
	}
	at, total := 0, 0
	for i, row := range s.rows {
		if !row.heading {
			total++
			if i <= s.cursor {
				at = total
			}
		}
	}
	return content, fmt.Sprintf("%d of %d", at, total)
}

func sideLine(row sideRow, width int) string {
	right := row.right
	if right != "" {
		right = " " + right + " "
	}
	room := max(1, width-1-ansi.StringWidth(right))
	return " " + fit(ansi.Truncate(row.text, room, "…"), room) + right
}

func (a App) tabs(n, width int) string {
	if n != reviewPanel || a.review == nil || a.review.page.Stack == nil {
		return ""
	}
	p := a.p
	tab := func(name string, on bool) string {
		if on {
			return p.Accent.Bold(true).Render(name)
		}
		return p.Muted.Render(name)
	}
	names := []string{"All"}
	for _, m := range a.review.state.Members {
		names = append(names, fmt.Sprintf("#%d", m.Pull))
	}
	var all []string
	active := ""
	for i, name := range names {
		on := (i == 0 && a.filter == 0) || name == fmt.Sprintf("#%d", a.filter)
		all = append(all, tab(name, on))
		if on {
			active = tab(name, true)
		}
	}
	sep := p.Border.Render(" - ")
	out := sep + strings.Join(all, sep)
	if ansi.StringWidth(out)+ansi.StringWidth(panelTitle(n, panelNames[n]))+3 > width {
		return sep + active
	}
	return out
}

func (a App) mainBox(r rect, focused bool) []string {
	p, in := a.p, r.inner()
	var content []string
	var shown extent
	switch {
	case a.loading:
		content = []string{"", " " + p.Muted.Render("Loading…")}
	case a.review == nil:
		content = []string{"", " " + p.Bad.Render(a.failed), "", " " + p.Muted.Render("R tries again; :q quits.")}
	case a.screen == screenDiff:
		content, shown = diffLines(*a.review, a.diff, in.h, a.filter, a.p, focused), a.diff.extent()
	case a.screen == screenDiagram && len(a.dg.level.Cards) == 0:
		content = []string{"", " " + p.Muted.Render("Nothing to draw: this change has no parts the diagram knows.")}
	case a.screen == screenDiagram:
		content, shown = a.dg.lines(*a.review, in.w, in.h, a.filter, a.styles)
	default:
		v := a.rv.view
		for i := v.top; i < v.top+in.h && i < len(v.lines); i++ {
			l := v.lines[i]
			text := l.text
			if l.item >= 0 && l.item == v.cursor && a.focus == mainPanel {
				text = p.highlight(text, in.w, focused)
			}
			content = append(content, text)
		}
		shown = v.extent()
	}
	return framed(content, shown, r.w, r.h, a.mainTitle(), "", a.placeWords(), focused, a.p)
}

func (a App) mainTitle() string {
	title := panelTitle(mainPanel, "Review")
	d := a.review
	switch {
	case d == nil:
	case a.screen == screenDiagram:
		title = panelTitle(mainPanel, a.diagramTitle())
	case a.screen == screenDiff:
		title = panelTitle(mainPanel, "Diff · "+componentTitle(*d, a.diff.component))
		if path := a.diff.file(); path != "" {
			title += " › " + path
		}
	}
	return title
}

func (a App) placeWords() string {
	if a.review == nil {
		return ""
	}
	switch a.screen {
	case screenDiff:
		if a.diff.selecting {
			return "VISUAL · " + a.selectionWords()
		}
		return a.diffPlace()
	case screenDiagram:
		return a.boxWords()
	}
	if v := a.rv.view; v.cursor < len(v.items) && a.focus == mainPanel {
		return itemWords(v.items[v.cursor])
	}
	return ""
}

func (a App) popup() []string {
	switch {
	case a.confirming != nil:
		c := a.confirming
		w := min(a.width-4, 56)
		body := wrap(fmt.Sprintf("Drop queued comment %d on #%d? Its text is gone once dropped.", c.comment, c.pull), w-4)
		for i, l := range body {
			body[i] = " " + a.p.Ink.Render(l)
		}
		return framed(body, extent{}, w, len(body)+2, fmt.Sprintf("Drop comment %d", c.comment), "", "y drop · any other key keeps it", true, a.p)
	case a.posting != nil:
		return a.postingBox()
	case a.menu != nil:
		return a.menu.box(a.width, min(a.height-3, a.menuRows()+2), a.p)
	}
	return nil
}

func (a App) barLine() string {
	p := a.p
	if a.prompting != "" {
		return fit(" "+a.prompt.View(), a.width)
	}
	right := ""
	switch {
	case a.notice != "":
		style := p.Ink
		if a.bad {
			style = p.Bad
		}
		right = style.Render(ansi.Truncate(a.notice, max(10, a.width*2/3), "…"))
	case a.keys.Typed() != "":
		right = p.Accent.Render(a.keys.Typed())
	case a.homeStatus() != "":
		right = p.Muted.Render(ansi.Truncate(a.homeStatus(), max(10, a.width*2/3), "…"))
	}
	hints := a.hints()
	room := a.width - ansi.StringWidth(right) - 3
	sep := p.Faint.Render(" | ")
	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = p.Muted.Render(h[1]+": ") + p.Accent.Render(h[0])
	}
	for len(parts) > 1 && ansi.StringWidth(strings.Join(parts, sep)) > room {
		parts = append(parts[:len(parts)-2], parts[len(parts)-1])
	}
	left := " " + ansi.Truncate(strings.Join(parts, sep), max(0, room), "…")
	if right != "" {
		right += " "
	}
	return spread(left, right, a.width)
}

func (a App) hints() [][2]string {
	keysHint := [2]string{"Space ?", "Keys"}
	switch {
	case a.confirming != nil:
		return [][2]string{{"y", "Drop"}, {"any other key", "Keep"}}
	case a.posting != nil:
		return a.postingHints()
	case a.keys.Pending() == "space":
		var hints [][2]string
		for _, k := range a.bindings().After("space") {
			hints = append(hints, [2]string{"Space " + k[0], leaderWords[k[1]]})
		}
		return hints
	case a.menu != nil && a.menu.finding:
		return [][2]string{{"type", "Narrow"}, {"Ctrl-n Ctrl-p", "Move"}, {"Enter", "Go"}, {"Esc", "Close"}}
	case a.menu != nil && a.menu.choosing():
		return [][2]string{{"j k", "Move"}, {"Enter", "Go"}, {"Esc", "Close"}}
	case a.menu != nil:
		return [][2]string{{"Esc", "Close"}}
	case a.screen == screenHome:
		return a.homeHints()
	case a.review == nil:
		return [][2]string{{"R", "Try again"}, {":q", "Quit"}}
	case a.screen == screenDiagram && a.dg.labels != nil:
		return [][2]string{{"a label", "Jump to its box"}, {"Esc", "Stop"}}
	}
	var hints [][2]string
	switch a.focus {
	case reviewPanel:
		hints = [][2]string{{"Enter", "Review"}, {"P", "Post"}, {"gx", "GitHub"}, {"2 3 4", "Files, notes, diagram"}}
	case filesPanel:
		hints = [][2]string{{"Enter", "Diff"}, {"za", "Fold"}, {"]p [p", "Pull request"}, {"1 3 4", "Panels"}}
	case notesPanel:
		hints = [][2]string{{"Enter", "Open"}, {"e", "Edit"}, {"dd", "Drop"}, {"P", "Post"}, {"1 2 4", "Panels"}}
	case diagramPanel:
		hints = [][2]string{{"Enter", "In"}, {"-", "Out"}, {"gd", "Follow"}, {"Space l", "Links"}, {"/", "Find"}, {"1 2 3", "Panels"}}
	default:
		hints = a.mainHints()
	}
	return append(hints, keysHint)
}

func (a App) mainHints() [][2]string {
	switch a.screen {
	case screenDiagram:
		return append(a.diagramHints(), [2]string{"Esc", "List"})
	case screenDiff:
		if a.diff.selecting {
			return [][2]string{{"c", "Comment"}, {"a", "Ask Claude"}, {"o", "Other end"}, {"Esc", "Stop"}}
		}
		return append(a.diffHints(), [2]string{"Esc", "List"})
	}
	if v := a.rv.view; v.cursor < len(v.items) && v.items[v.cursor].pull != 0 {
		return [][2]string{{"j k", "Move"}, {"Enter", "Open"}, {"P", "Post"}, {"e dd", "Edit, drop"}, {"]] [[", "Sections"}, {"Esc", "List"}}
	}
	return [][2]string{{"j k", "Move"}, {"Enter", "Open"}, {"]] [[", "Sections"}, {"za", "Fold"}, {"Esc", "List"}}
}

func (a App) selectionWords() string {
	at, _, ok := a.selected()
	if !ok {
		return "a selection stays in one file"
	}
	words := linesWords(at.Side, at.Line, at.StartLine, at.StartSide)
	if a.where.key == linesKey(at) && a.where.words != "" {
		return words + " · " + a.where.words
	}
	return words
}

func (a App) diffHints() [][2]string {
	if n, r, ok := a.noteAt(); ok {
		switch {
		case n.kind == "queued":
			return [][2]string{{"e", "Edit"}, {"dd", "Drop"}, {"za", "Fold"}, {"]n [n", "Notes"}, {"-", "Back"}}
		case n.kind == "conversation" && r.part == "claude":
			return [][2]string{{"c", "Comment with this answer"}, {"a", "Follow-up"}, {"za", "Fold"}, {"]n [n", "Notes"}}
		case n.kind == "conversation":
			return [][2]string{{"a", "Follow-up"}, {"za", "Fold"}, {"]n [n", "Notes"}, {"-", "Back"}}
		}
		return [][2]string{{"za", "Fold"}, {"gx", "On GitHub"}, {"]n [n", "Notes"}, {"-", "Back"}}
	}
	return [][2]string{{"j k", "Move"}, {"]c [c", "Hunks"}, {"V", "Select"}, {"c", "Comment"}, {"a", "Ask Claude"}, {"/", "Search"}, {"-", "Back"}}
}

func spread(left, right string, width int) string {
	gap := width - ansi.StringWidth(left) - ansi.StringWidth(right)
	if gap < 1 {
		return fit(left, width)
	}
	return left + strings.Repeat(" ", gap) + right
}

func (a App) diffPlace() string {
	v := a.diff
	if v.cursor >= len(v.rows) {
		return ""
	}
	r := v.rows[v.cursor]
	if n, ok := noteOf(*a.review, r.path, r.key); ok && (r.kind == "notehead" || r.kind == "notetext") {
		return noteWords(n)
	}
	if r.kind == "block" {
		b, start := r.block, 0
		if b.StartLine != b.Line {
			start = b.StartLine
		}
		return linesWords(b.Side, b.Line, start, "") + " · " + map[string]string{"add": "added", "del": "removed", "same": "unchanged"}[b.State]
	}
	side, line, ok := r.at()
	if !ok {
		return map[string]string{"file": "file header", "hunk": "hunk header", "gap": "unchanged lines", "note": ""}[r.kind]
	}
	words := linesWords(side, line, 0, "")
	if w := r.words(); w != "" {
		words += " · " + w
	}
	return words
}

func itemWords(it item) string {
	switch it.kind {
	case "section":
		return it.title
	case "comment":
		return fmt.Sprintf("queued comment %d", it.comment)
	case "link", "thread", "conversation":
		if it.place != nil {
			return fmt.Sprintf("%s %s", it.place.Path, linesWords(it.place.Side, it.place.Line, it.place.StartLine, it.place.StartSide))
		}
		return it.kind
	case "component", "pull":
		return it.title
	}
	return ""
}

func componentTitle(d reviewData, id string) string {
	for _, c := range d.page.Model.Components {
		if c.ID == id {
			return c.Title
		}
	}
	return id
}
