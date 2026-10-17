package gui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
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
		rows[i] = utils.Fit(rows[i], a.width)
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
	content, place := sideContent(a.c.Tr, a.sides[n], r.inner(), focused, a.p)
	if n == reviewPanel && a.growFrom != 0 {
		place = a.c.Tr.Visual + " · " + place
	}
	return framed(content, extent{}, r.w, r.h, panelTitle(n, panelNames(a.c.Tr)[n]), a.tabs(n, r.inner().w), place, focused, a.p)
}

func sideContent(tr *i18n.TranslationSet, s sidePanel, in rect, focused bool, p palette) ([]string, string) {
	var content []string
	for i := s.top; i < s.top+in.h && i < len(s.rows); i++ {
		row := s.rows[i]
		line := sideLine(row, in.w)
		switch {
		case i == s.cursor && !row.heading && (focused || !row.lit):
			line = p.highlight(line, in.w, focused)
		case row.lit:
			line = p.lit(line, in.w)
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
	return content, ofTotal(tr, at, total)
}

func ofTotal(tr *i18n.TranslationSet, at, total int) string {
	return utils.ResolvePlaceholderString(tr.OfTotal, map[string]string{"at": utils.Itoa(at), "total": utils.Itoa(total)})
}

func sideLine(row sideRow, width int) string {
	right := row.right
	if right != "" {
		right = " " + right + " "
	}
	room := max(1, width-1-ansi.StringWidth(right))
	text := row.text
	if row.title != "" {
		text += row.titleStyle.Render(utils.FitTitle(row.title, room-ansi.StringWidth(text)))
	}
	return " " + utils.Fit(ansi.Truncate(text, room, "…"), room) + right
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
	names, on := []string{a.c.Tr.All}, []bool{len(a.filter) == 0}
	for _, m := range a.review.state.Members {
		names = append(names, fmt.Sprintf("#%d", m.Pull))
		on = append(on, containsInt(a.filter, m.Pull))
	}
	var all, active []string
	for i, name := range names {
		all = append(all, tab(name, on[i]))
		if on[i] {
			active = append(active, tab(name, true))
		}
	}
	sep := p.Border.Render(" - ")
	room := width - ansi.StringWidth(panelTitle(n, panelNames(a.c.Tr)[n])) - 3
	fits := []string{sep + strings.Join(all, sep), sep + strings.Join(active, sep)}
	if p.plain {
		fits = fits[1:]
	}
	if len(active) > 2 {
		fits = append(fits, sep+active[0]+p.Muted.Render("-")+active[len(active)-1])
	}
	for _, out := range fits[:len(fits)-1] {
		if ansi.StringWidth(out) <= room {
			return out
		}
	}
	return fits[len(fits)-1]
}

func (a App) mainBox(r rect, focused bool) []string {
	p, in, tr := a.p, r.inner(), a.c.Tr
	var content []string
	var shown extent
	switch {
	case a.loading:
		content = []string{"", " " + p.Muted.Render(tr.Loading)}
	case a.review == nil:
		content = []string{"", " " + p.Bad.Render(a.failed), "", " " + p.Muted.Render(tr.TryAgainOrQuit)}
	case a.screen == screenDiff:
		content, shown = diffLines(tr, *a.review, a.diff, in.h, a.filter, a.p, focused), a.diff.extent()
	case a.screen == screenDiagram && len(a.dg.level.Cards) == 0:
		content = []string{"", " " + p.Muted.Render(tr.NothingToDraw)}
	case a.screen == screenDiagram:
		content, shown = a.dg.lines(tr, *a.review, in.w, in.h, a.filter, a.styles)
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
	return framed(content, shown, r.w, r.h, a.mainTitle(r.w-4), "", a.placeWords(r.w-4), focused, a.p)
}

func (a App) mainTitle(width int) string {
	title, path := panelTitle(mainPanel, a.c.Tr.ReviewTitle), ""
	d := a.review
	switch {
	case d == nil:
	case a.screen == screenDiagram:
		head, in := a.diagramTitle()
		title, path = panelTitle(mainPanel, head), in
	case a.screen == screenDiff:
		title = panelTitle(mainPanel, a.c.Tr.DiffTitle+" · "+componentTitle(*d, a.diff.component))
		if path = a.diff.file(); path != "" {
			title += " › "
		}
	}
	return utils.FitPathIn(title, path, "", width)
}

func (a App) placeWords(width int) string {
	if a.review == nil {
		return ""
	}
	switch a.screen {
	case screenDiff:
		if a.diff.selecting {
			return a.c.Tr.Visual + " · " + a.selectionWords()
		}
		return a.diffPlace()
	case screenDiagram:
		return a.boxWords(width)
	}
	if v := a.rv.view; v.cursor < len(v.items) && a.focus == mainPanel {
		return itemWords(a.c.Tr, v.items[v.cursor], width)
	}
	return ""
}

func (a App) popup() []string {
	switch {
	case a.confirming != nil:
		c, tr := a.confirming, a.c.Tr
		w := min(a.width-4, 56)
		body := utils.Wrap(utils.ResolvePlaceholderString(tr.DropQuestion, map[string]string{"comment": utils.Itoa(c.comment), "pull": utils.Itoa(c.pull)}), w-4)
		for i, l := range body {
			body[i] = " " + a.p.Ink.Render(l)
		}
		title := utils.ResolvePlaceholderString(tr.DropTitle, map[string]string{"comment": utils.Itoa(c.comment)})
		return framed(body, extent{}, w, len(body)+2, title, "", tr.DropFooter, true, a.p)
	case a.posting != nil:
		return a.postingBox()
	case a.menu != nil:
		return a.menu.box(a.c.Tr, a.width, min(a.height-3, a.menuRows()+2), a.p)
	}
	return nil
}

func (a App) barLine() string {
	p := a.p
	if a.prompting != "" {
		return utils.Fit(" "+a.prompt.View(), a.width)
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
	tr := a.c.Tr
	keysHint := [2]string{"Space ?", tr.Keys}
	switch {
	case a.confirming != nil:
		return [][2]string{{"y", tr.Drop}, {tr.AnyOtherKey, tr.Keep}}
	case a.posting != nil:
		return a.postingHints()
	case a.keys.Pending() == "space":
		var hints [][2]string
		for _, k := range a.bindings().After("space") {
			hints = append(hints, [2]string{"Space " + k[0], leaderWords(tr)[k[1]]})
		}
		return hints
	case a.menu != nil && a.menu.finding:
		return [][2]string{{tr.TypeKey, tr.Narrow}, {"Ctrl-n Ctrl-p", tr.Move}, {"Enter", tr.Go}, {"Esc", tr.Close}}
	case a.menu != nil && a.menu.choosing():
		return [][2]string{{"j k", tr.Move}, {"Enter", tr.Go}, {"Esc", tr.Close}}
	case a.menu != nil:
		return [][2]string{{"Esc", tr.Close}}
	case a.screen == screenHome:
		return a.homeHints()
	case a.review == nil:
		return [][2]string{{"R", tr.TryAgain}, {":q", tr.Quit}}
	case a.screen == screenDiagram && a.dg.labels != nil:
		return [][2]string{{tr.ALabelKey, tr.JumpToBox}, {"Esc", tr.Stop}}
	}
	var hints [][2]string
	switch a.focus {
	case reviewPanel:
		hints = a.reviewPanelHints()
	case filesPanel:
		hints = [][2]string{{"Enter", tr.Diff}, {"za", tr.Fold}, {"]p [p", tr.PullRequest}, {"h l", tr.Panels}}
	case notesPanel:
		hints = [][2]string{{"Enter", tr.Open}, {"e", tr.Edit}, {"dd", tr.Drop}, {"P", tr.Post}, {"h l", tr.Panels}}
	case diagramPanel:
		hints = [][2]string{{"h j k l", tr.Move}, {"Enter", tr.In}, {"-", tr.Out}, {"1 2 3", tr.Panels}, {"gd", tr.Follow}, {"Space l", tr.Links}, {"/", tr.Find}}
	default:
		hints = a.mainHints()
	}
	return append(hints, keysHint)
}

func (a App) reviewPanelHints() [][2]string {
	tr := a.c.Tr
	panels := [2]string{"h l", tr.Panels}
	switch {
	case a.growFrom != 0:
		return [][2]string{{"j k", tr.GrowShrink}, {"Esc", tr.All}, panels}
	case a.review.page.Stack == nil:
		return [][2]string{{"Enter", tr.Review}, {"P", tr.Post}, {"gx", tr.GitHub}, panels}
	case len(a.filter) > 0:
		return [][2]string{{"Enter", tr.Review}, {"v", tr.Highlight}, {"Esc", tr.All}, {"P", tr.Post}, {"gx", tr.GitHub}, panels}
	}
	return [][2]string{{"Enter", tr.Review}, {"v", tr.Highlight}, {"P", tr.Post}, {"gx", tr.GitHub}, panels}
}

func (a App) mainHints() [][2]string {
	tr := a.c.Tr
	switch a.screen {
	case screenDiagram:
		return append(a.diagramHints(), [2]string{"Esc", tr.List})
	case screenDiff:
		if a.diff.selecting {
			return [][2]string{{"c", tr.Comment}, {"a", tr.AskClaude}, {"o", tr.OtherEnd}, {"Esc", tr.Stop}}
		}
		return append(a.diffHints(), [2]string{"Esc", tr.List})
	}
	if v := a.rv.view; v.cursor < len(v.items) && v.items[v.cursor].pull != 0 {
		return [][2]string{{"j k", tr.Move}, {"Enter", tr.Open}, {"P", tr.Post}, {"e dd", tr.EditDrop}, {"]] [[", tr.Sections}, {"Esc", tr.List}}
	}
	return [][2]string{{"j k", tr.Move}, {"Enter", tr.Open}, {"]] [[", tr.Sections}, {"za", tr.Fold}, {"Esc", tr.List}}
}

func (a App) selectionWords() string {
	at, _, ok := a.selected()
	if !ok {
		return a.c.Tr.SelectionInOneFile
	}
	words := linesWords(a.c.Tr, at.Side, at.Line, at.StartLine, at.StartSide)
	if a.where.key == linesKey(at) && a.where.words != "" {
		return words + " · " + a.where.words
	}
	return words
}

func (a App) diffHints() [][2]string {
	tr := a.c.Tr
	if n, r, ok := a.noteAt(); ok {
		switch {
		case n.kind == "queued":
			return [][2]string{{"e", tr.Edit}, {"dd", tr.Drop}, {"za", tr.Fold}, {"]n [n", tr.Notes}, {"-", tr.Back}}
		case n.kind == "conversation" && r.part == "claude":
			return [][2]string{{"c", tr.CommentWithAnswer}, {"a", tr.FollowUp}, {"za", tr.Fold}, {"]n [n", tr.Notes}}
		case n.kind == "conversation":
			return [][2]string{{"a", tr.FollowUp}, {"za", tr.Fold}, {"]n [n", tr.Notes}, {"-", tr.Back}}
		}
		return [][2]string{{"za", tr.Fold}, {"gx", tr.OnGitHub}, {"]n [n", tr.Notes}, {"-", tr.Back}}
	}
	return [][2]string{{"j k", tr.Move}, {"]c [c", tr.Hunks}, {"V", tr.Select}, {"c", tr.Comment}, {"a", tr.AskClaude}, {"/", tr.Search}, {"-", tr.Back}}
}

func spread(left, right string, width int) string {
	gap := width - ansi.StringWidth(left) - ansi.StringWidth(right)
	if gap < 1 {
		return utils.Fit(left, width)
	}
	return left + strings.Repeat(" ", gap) + right
}

func (a App) diffPlace() string {
	v := a.diff
	if v.cursor >= len(v.rows) {
		return ""
	}
	r, tr := v.rows[v.cursor], a.c.Tr
	if n, ok := noteOf(*a.review, r.path, r.key); ok && (r.kind == "notehead" || r.kind == "notetext") {
		return noteWords(tr, n)
	}
	if r.kind == "block" {
		b, start := r.block, 0
		if b.StartLine != b.Line {
			start = b.StartLine
		}
		return linesWords(tr, b.Side, b.Line, start, "") + " · " + map[string]string{"add": tr.Added, "del": tr.Removed, "same": tr.Unchanged}[b.State]
	}
	side, line, ok := r.at()
	if !ok {
		return map[string]string{"file": tr.FileHeader, "hunk": tr.HunkHeader, "gap": tr.UnchangedLines, "note": ""}[r.kind]
	}
	words := linesWords(tr, side, line, 0, "")
	if w := r.words(); w != "" {
		words += " · " + w
	}
	return words
}

func itemWords(tr *i18n.TranslationSet, it item, width int) string {
	switch it.kind {
	case "section":
		return it.title
	case "comment":
		return utils.ResolvePlaceholderString(tr.QueuedComment, map[string]string{"comment": utils.Itoa(it.comment)})
	case "link", "thread", "conversation":
		if it.place != nil {
			return utils.FitPathIn("", it.place.Path, " "+linesWords(tr, it.place.Side, it.place.Line, it.place.StartLine, it.place.StartSide), width)
		}
		return map[string]string{"link": tr.LinkNoun, "thread": tr.ThreadNoun, "conversation": tr.ConversationNoun}[it.kind]
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
