package gui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// View draws the screen: the panels on the left, the main panel on the
// right, any popup over them, and a bar of the keys that matter here. The
// home screen has a panel on the left for each repository; a review has its
// four.
func (a App) View() string {
	if a.Width == 0 || a.Height == 0 {
		return ""
	}
	rows := make([]string, max(0, a.Height-1))
	place := func(r views.Rect, lines []string) {
		for i, line := range lines {
			if y := r.Y + i; y >= 0 && y < len(rows) {
				rows[y] += line
			}
		}
	}
	if a.Screen() == context.ScreenHome {
		l := a.homeLayout()
		for i, r := range l.panels {
			if r.W > 2 && r.H > 0 {
				place(r, a.homePanelBox(i, r, a.Home.Focus == i && !a.popped()))
			}
		}
		if r := l.main; r.Shown() {
			place(r, a.homeMainBox(r))
		}
	} else {
		l := a.layout()
		for n := context.ReviewPanel; n <= context.SideCount; n++ {
			if r := l.sides[n]; r.W > 2 && r.H > 0 {
				place(r, a.sideBox(n, r, a.Focus() == n && !a.popped()))
			}
		}
		if r := l.main; r.Shown() {
			place(r, a.mainBox(r, a.Focus() == context.MainPanel && !a.popped()))
		}
	}
	for i := range rows {
		rows[i] = utils.Fit(rows[i], a.Width)
	}
	if box := a.popup(); len(box) > 0 {
		w := ansi.StringWidth(box[0])
		rows = views.Overlay(rows, box, max(0, (a.Width-w)/2), max(0, (len(rows)-len(box))/2), a.c.Theme)
	}
	return strings.Join(append(rows, a.barLine()), "\n")
}

func (a App) sideBox(n int, r views.Rect, focused bool) []string {
	s := a.Side(n)
	content, place := presentation.SideContent(a.c.Tr, a.c.Theme, s.Rows, s.Cursor, s.Top, r.Inner(), focused)
	if n == context.ReviewPanel && a.GrowFrom != 0 {
		place = a.c.Tr.Visual + " · " + place
	}
	return views.Framed(content, views.Extent{}, r.W, r.H, views.PanelTitle(n, panelNames(a.c.Tr)[n]), a.tabs(n, r.Inner().W), place, focused, a.c.Theme)
}

func (a App) tabs(n, width int) string {
	if n != context.ReviewPanel || a.Review == nil || a.Review.Page.Stack == nil {
		return ""
	}
	room := width - ansi.StringWidth(views.PanelTitle(n, panelNames(a.c.Tr)[n])) - 3
	return presentation.Tabs(a.c.Tr, a.c.Theme, *a.Review, a.Filter, room)
}

func (a App) mainBox(r views.Rect, focused bool) []string {
	th, in, tr := a.c.Theme, r.Inner(), a.c.Tr
	var content []string
	var shown views.Extent
	switch {
	case a.Loading:
		content = []string{"", " " + th.Muted.Render(tr.Loading)}
	case a.Review == nil:
		content = []string{"", " " + th.Bad.Render(a.Failed), "", " " + th.Muted.Render(tr.TryAgainOrQuit)}
	case a.Screen() == context.ScreenDiff:
		content, shown = presentation.DiffLines(tr, th, *a.Review, a.Diff.DiffView, in.H, a.Filter, focused), a.Diff.Extent()
	case a.Screen() == context.ScreenDiagram && len(a.Diagram.Level.Cards) == 0:
		content = []string{"", " " + th.Muted.Render(tr.NothingToDraw)}
	case a.Screen() == context.ScreenDiagram:
		content, shown = a.Diagram.Lines(tr, *a.Review, in.W, in.H, a.Filter, a.styles)
	default:
		v := a.ReviewView.View
		for i := v.Top; i < v.Top+in.H && i < len(v.Lines); i++ {
			l := v.Lines[i]
			text := l.Text
			if l.Item >= 0 && l.Item == v.Cursor && a.Focus() == context.MainPanel {
				text = th.Selected(text, in.W, focused)
			}
			content = append(content, text)
		}
		shown = v.Extent()
	}
	return views.Framed(content, shown, r.W, r.H, a.mainTitle(r.W-4), "", a.placeWords(r.W-4), focused, a.c.Theme)
}

func (a App) mainTitle(width int) string {
	title, path := views.PanelTitle(context.MainPanel, a.c.Tr.ReviewTitle), ""
	d := a.Review
	switch {
	case d == nil:
	case a.Screen() == context.ScreenDiagram:
		head, in := presentation.DiagramTitle(a.c.Tr, *d, a.Diagram.Level)
		title, path = views.PanelTitle(context.MainPanel, head), in
	case a.Screen() == context.ScreenDiff:
		title = views.PanelTitle(context.MainPanel, a.c.Tr.DiffTitle+" · "+presentation.ComponentTitle(*d, a.Diff.Component))
		if path = a.Diff.File(); path != "" {
			title += " › "
		}
	}
	return utils.FitPathIn(title, path, "", width)
}

func (a App) placeWords(width int) string {
	if a.Review == nil {
		return ""
	}
	switch a.Screen() {
	case context.ScreenDiff:
		if a.Diff.Selecting {
			return a.c.Tr.Visual + " · " + a.selectionWords()
		}
		if v := a.Diff; v.Cursor < len(v.Rows) {
			return presentation.DiffRowWords(a.c.Tr, *a.Review, v.Rows[v.Cursor])
		}
		return ""
	case context.ScreenDiagram:
		if card, ok := a.Diagram.Card(); ok {
			return presentation.BoxWords(a.c.Tr, *a.Review, card, width)
		}
		return a.c.Tr.NothingToDrawHere
	}
	if v := a.ReviewView.View; v.Cursor < len(v.Items) && a.Focus() == context.MainPanel {
		return presentation.ItemWords(a.c.Tr, v.Items[v.Cursor], width)
	}
	return ""
}

func (a App) popup() []string {
	switch {
	case a.Has(context.ConfirmationKey):
		return presentation.DropBox(a.c.Tr, a.c.Theme, a.Confirmation.Comment, a.Confirmation.Pull, min(a.Width-4, 56))
	case a.Has(context.PostingKey):
		return a.postingBox()
	case a.menuOpen():
		return a.Menu.Box(a.c.Tr, a.c.Theme, a.Width, min(a.Height-3, a.menuRows()+2))
	}
	return nil
}

func (a App) barLine() string {
	th := a.c.Theme
	if a.Has(context.PromptKey) {
		return utils.Fit(" "+a.Prompt.Input.View(), a.Width)
	}
	right := ""
	switch {
	case a.Notice != "":
		style := th.Text
		if a.Bad {
			style = th.Bad
		}
		right = style.Render(ansi.Truncate(a.Notice, max(10, a.Width*2/3), "…"))
	case a.Keys.Typed() != "":
		right = th.Accent.Render(a.Keys.Typed())
	case a.homeStatus() != "":
		right = th.Muted.Render(ansi.Truncate(a.homeStatus(), max(10, a.Width*2/3), "…"))
	}
	hints := a.hints()
	room := a.Width - ansi.StringWidth(right) - 3
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
	return utils.Spread(left, right, a.Width)
}

func (a App) hints() [][2]string {
	tr := a.c.Tr
	keysHint := [2]string{"Space ?", tr.Keys}
	switch {
	case a.Has(context.ConfirmationKey):
		return [][2]string{{"y", tr.Drop}, {tr.AnyOtherKey, tr.Keep}}
	case a.Has(context.PostingKey):
		return a.postingHints()
	case a.Keys.Pending() == "space":
		var hints [][2]string
		for _, k := range a.bindings().After("space") {
			hints = append(hints, [2]string{"Space " + k[0], leaderWords(tr)[k[1]]})
		}
		return hints
	case a.menuOpen() && a.Menu.Finding:
		return [][2]string{{tr.TypeKey, tr.Narrow}, {"Ctrl-n Ctrl-p", tr.Move}, {"Enter", tr.Go}, {"Esc", tr.Close}}
	case a.menuOpen() && a.Menu.Choosing():
		return [][2]string{{"j k", tr.Move}, {"Enter", tr.Go}, {"Esc", tr.Close}}
	case a.menuOpen():
		return [][2]string{{"Esc", tr.Close}}
	case a.Screen() == context.ScreenHome:
		return a.homeHints()
	case a.Review == nil:
		return [][2]string{{"R", tr.TryAgain}, {":q", tr.Quit}}
	case a.Screen() == context.ScreenDiagram && a.Has(context.LabelsKey):
		return [][2]string{{tr.ALabelKey, tr.JumpToBox}, {"Esc", tr.Stop}}
	}
	var hints [][2]string
	switch a.Focus() {
	case context.ReviewPanel:
		hints = a.reviewPanelHints()
	case context.FilesPanel:
		hints = [][2]string{{"Enter", tr.Diff}, {"za", tr.Fold}, {"]p [p", tr.PullRequest}, {"h l", tr.Panels}}
	case context.NotesPanel:
		hints = [][2]string{{"Enter", tr.Open}, {"e", tr.Edit}, {"dd", tr.Drop}, {"P", tr.Post}, {"h l", tr.Panels}}
	case context.DiagramPanel:
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
	case a.GrowFrom != 0:
		return [][2]string{{"j k", tr.GrowShrink}, {"Esc", tr.All}, panels}
	case a.Review.Page.Stack == nil:
		return [][2]string{{"Enter", tr.Review}, {"P", tr.Post}, {"gx", tr.GitHub}, panels}
	case len(a.Filter) > 0:
		return [][2]string{{"Enter", tr.Review}, {"v", tr.Highlight}, {"Esc", tr.All}, {"P", tr.Post}, {"gx", tr.GitHub}, panels}
	}
	return [][2]string{{"Enter", tr.Review}, {"v", tr.Highlight}, {"P", tr.Post}, {"gx", tr.GitHub}, panels}
}

func (a App) mainHints() [][2]string {
	tr := a.c.Tr
	switch a.Screen() {
	case context.ScreenDiagram:
		return append(a.diagramHints(), [2]string{"Esc", tr.List})
	case context.ScreenDiff:
		if a.Diff.Selecting {
			return [][2]string{{"c", tr.Comment}, {"a", tr.AskClaude}, {"o", tr.OtherEnd}, {"Esc", tr.Stop}}
		}
		return append(a.diffHints(), [2]string{"Esc", tr.List})
	}
	if v := a.ReviewView.View; v.Cursor < len(v.Items) && v.Items[v.Cursor].Pull != 0 {
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
	if a.Where.Key == linesKey(at) && a.Where.Words != "" {
		return words + " · " + a.Where.Words
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
