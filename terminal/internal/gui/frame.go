package gui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
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
	place := func(r views.Rect, lines []string) {
		for i, line := range lines {
			if y := r.Y + i; y >= 0 && y < len(rows) {
				rows[y] += line
			}
		}
	}
	if a.screen == screenHome {
		l := a.homeLayout()
		for i, r := range l.panels {
			if r.W > 2 && r.H > 0 {
				place(r, a.homePanelBox(i, r, a.home.focus == i && !a.popped()))
			}
		}
		if r := l.main; r.Shown() {
			place(r, a.homeMainBox(r))
		}
	} else {
		l := a.layout()
		for n := reviewPanel; n <= sideCount; n++ {
			if r := l.sides[n]; r.W > 2 && r.H > 0 {
				place(r, a.sideBox(n, r, a.focus == n && !a.popped()))
			}
		}
		if r := l.main; r.Shown() {
			place(r, a.mainBox(r, a.focus == mainPanel && !a.popped()))
		}
	}
	for i := range rows {
		rows[i] = utils.Fit(rows[i], a.width)
	}
	if box := a.popup(); len(box) > 0 {
		w := ansi.StringWidth(box[0])
		rows = views.Overlay(rows, box, max(0, (a.width-w)/2), max(0, (len(rows)-len(box))/2), a.c.Theme)
	}
	return strings.Join(append(rows, a.barLine()), "\n")
}

func (a App) popped() bool {
	return a.menu != nil || a.posting != nil || a.confirming != nil
}

func (a App) sideBox(n int, r views.Rect, focused bool) []string {
	s := a.sides[n]
	content, place := presentation.SideContent(a.c.Tr, a.c.Theme, s.rows, s.cursor, s.top, r.Inner(), focused)
	if n == reviewPanel && a.growFrom != 0 {
		place = a.c.Tr.Visual + " · " + place
	}
	return views.Framed(content, views.Extent{}, r.W, r.H, views.PanelTitle(n, panelNames(a.c.Tr)[n]), a.tabs(n, r.Inner().W), place, focused, a.c.Theme)
}

func (a App) tabs(n, width int) string {
	if n != reviewPanel || a.review == nil || a.review.Page.Stack == nil {
		return ""
	}
	room := width - ansi.StringWidth(views.PanelTitle(n, panelNames(a.c.Tr)[n])) - 3
	return presentation.Tabs(a.c.Tr, a.c.Theme, *a.review, a.filter, room)
}

func (a App) mainBox(r views.Rect, focused bool) []string {
	th, in, tr := a.c.Theme, r.Inner(), a.c.Tr
	var content []string
	var shown views.Extent
	switch {
	case a.loading:
		content = []string{"", " " + th.Muted.Render(tr.Loading)}
	case a.review == nil:
		content = []string{"", " " + th.Bad.Render(a.failed), "", " " + th.Muted.Render(tr.TryAgainOrQuit)}
	case a.screen == screenDiff:
		content, shown = presentation.DiffLines(tr, th, *a.review, a.diff.DiffView, in.H, a.filter, focused), a.diff.extent()
	case a.screen == screenDiagram && len(a.dg.Level.Cards) == 0:
		content = []string{"", " " + th.Muted.Render(tr.NothingToDraw)}
	case a.screen == screenDiagram:
		content, shown = a.dg.lines(tr, *a.review, in.W, in.H, a.filter, a.styles)
	default:
		v := a.rv.view
		for i := v.top; i < v.top+in.H && i < len(v.lines); i++ {
			l := v.lines[i]
			text := l.Text
			if l.Item >= 0 && l.Item == v.cursor && a.focus == mainPanel {
				text = th.Selected(text, in.W, focused)
			}
			content = append(content, text)
		}
		shown = v.extent()
	}
	return views.Framed(content, shown, r.W, r.H, a.mainTitle(r.W-4), "", a.placeWords(r.W-4), focused, a.c.Theme)
}

func (a App) mainTitle(width int) string {
	title, path := views.PanelTitle(mainPanel, a.c.Tr.ReviewTitle), ""
	d := a.review
	switch {
	case d == nil:
	case a.screen == screenDiagram:
		head, in := presentation.DiagramTitle(a.c.Tr, *d, a.dg.Level)
		title, path = views.PanelTitle(mainPanel, head), in
	case a.screen == screenDiff:
		title = views.PanelTitle(mainPanel, a.c.Tr.DiffTitle+" · "+presentation.ComponentTitle(*d, a.diff.Component))
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
		if a.diff.Selecting {
			return a.c.Tr.Visual + " · " + a.selectionWords()
		}
		if v := a.diff; v.Cursor < len(v.Rows) {
			return presentation.DiffRowWords(a.c.Tr, *a.review, v.Rows[v.Cursor])
		}
		return ""
	case screenDiagram:
		if card, ok := a.dg.Card(); ok {
			return presentation.BoxWords(a.c.Tr, *a.review, card, width)
		}
		return a.c.Tr.NothingToDrawHere
	}
	if v := a.rv.view; v.cursor < len(v.items) && a.focus == mainPanel {
		return presentation.ItemWords(a.c.Tr, v.items[v.cursor], width)
	}
	return ""
}

func (a App) popup() []string {
	switch {
	case a.confirming != nil:
		return presentation.DropBox(a.c.Tr, a.c.Theme, a.confirming.comment, a.confirming.pull, min(a.width-4, 56))
	case a.posting != nil:
		return a.postingBox()
	case a.menu != nil:
		return a.menu.Box(a.c.Tr, a.c.Theme, a.width, min(a.height-3, a.menuRows()+2))
	}
	return nil
}

func (a App) barLine() string {
	th := a.c.Theme
	if a.prompting != "" {
		return utils.Fit(" "+a.prompt.View(), a.width)
	}
	right := ""
	switch {
	case a.notice != "":
		style := th.Text
		if a.bad {
			style = th.Bad
		}
		right = style.Render(ansi.Truncate(a.notice, max(10, a.width*2/3), "…"))
	case a.keys.Typed() != "":
		right = th.Accent.Render(a.keys.Typed())
	case a.homeStatus() != "":
		right = th.Muted.Render(ansi.Truncate(a.homeStatus(), max(10, a.width*2/3), "…"))
	}
	hints := a.hints()
	room := a.width - ansi.StringWidth(right) - 3
	sep := th.Faint.Render(" | ")
	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = th.Muted.Render(h[1]+": ") + th.Accent.Render(h[0])
	}
	for len(parts) > 1 && ansi.StringWidth(strings.Join(parts, sep)) > room {
		parts = append(parts[:len(parts)-2], parts[len(parts)-1])
	}
	left := " " + ansi.Truncate(strings.Join(parts, sep), max(0, room), "…")
	if right != "" {
		right += " "
	}
	return utils.Spread(left, right, a.width)
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
	case a.menu != nil && a.menu.Finding:
		return [][2]string{{tr.TypeKey, tr.Narrow}, {"Ctrl-n Ctrl-p", tr.Move}, {"Enter", tr.Go}, {"Esc", tr.Close}}
	case a.menu != nil && a.menu.Choosing():
		return [][2]string{{"j k", tr.Move}, {"Enter", tr.Go}, {"Esc", tr.Close}}
	case a.menu != nil:
		return [][2]string{{"Esc", tr.Close}}
	case a.screen == screenHome:
		return a.homeHints()
	case a.review == nil:
		return [][2]string{{"R", tr.TryAgain}, {":q", tr.Quit}}
	case a.screen == screenDiagram && a.dg.Labels != nil:
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
	case a.review.Page.Stack == nil:
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
		if a.diff.Selecting {
			return [][2]string{{"c", tr.Comment}, {"a", tr.AskClaude}, {"o", tr.OtherEnd}, {"Esc", tr.Stop}}
		}
		return append(a.diffHints(), [2]string{"Esc", tr.List})
	}
	if v := a.rv.view; v.cursor < len(v.items) && v.items[v.cursor].Pull != 0 {
		return [][2]string{{"j k", tr.Move}, {"Enter", tr.Open}, {"P", tr.Post}, {"e dd", tr.EditDrop}, {"]] [[", tr.Sections}, {"Esc", tr.List}}
	}
	return [][2]string{{"j k", tr.Move}, {"Enter", tr.Open}, {"]] [[", tr.Sections}, {"za", tr.Fold}, {"Esc", tr.List}}
}

func (a App) selectionWords() string {
	at, _, ok := a.selected()
	if !ok {
		return a.c.Tr.SelectionInOneFile
	}
	words := presentation.LinesWords(a.c.Tr, at.Side, at.Line, at.StartLine, at.StartSide)
	if a.where.key == linesKey(at) && a.where.words != "" {
		return words + " · " + a.where.words
	}
	return words
}

func (a App) diffHints() [][2]string {
	tr := a.c.Tr
	if n, r, ok := a.noteAt(); ok {
		switch {
		case n.Kind == "queued":
			return [][2]string{{"e", tr.Edit}, {"dd", tr.Drop}, {"za", tr.Fold}, {"]n [n", tr.Notes}, {"-", tr.Back}}
		case n.Kind == "conversation" && r.Part == "claude":
			return [][2]string{{"c", tr.CommentWithAnswer}, {"a", tr.FollowUp}, {"za", tr.Fold}, {"]n [n", tr.Notes}}
		case n.Kind == "conversation":
			return [][2]string{{"a", tr.FollowUp}, {"za", tr.Fold}, {"]n [n", tr.Notes}, {"-", tr.Back}}
		}
		return [][2]string{{"za", tr.Fold}, {"gx", tr.OnGitHub}, {"]n [n", tr.Notes}, {"-", tr.Back}}
	}
	return [][2]string{{"j k", tr.Move}, {"]c [c", tr.Hunks}, {"V", tr.Select}, {"c", tr.Comment}, {"a", tr.AskClaude}, {"/", tr.Search}, {"-", tr.Back}}
}
