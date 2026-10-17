package gui

import (
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func (a App) mainBox(r views.Rect, focused bool) []string {
	th, in, tr := a.c.Theme, r.Inner(), a.c.Tr
	var content []string
	var shown views.Extent
	switch {
	case a.Loading:
		content = []string{"", " " + th.Muted.Render(tr.Loading)}
	case a.Review == nil:
		labels := a.controllers.Labels(a.Place())
		again := utils.ResolvePlaceholderString(tr.TryAgainOrQuit, map[string]string{"refresh": labels.Of("refresh"), "command": labels.Of("command")})
		content = []string{"", " " + th.Bad.Render(a.Failed), "", " " + th.Muted.Render(again)}
	case a.Screen() == context.ScreenDiff:
		content, shown = presentation.DiffLines(tr, th, a.controllers.Labels(context.DiffKey), *a.Review, a.Diff.DiffView, in.H, a.Filter, focused), a.Diff.Extent()
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

func (a App) selectionWords() string {
	at, _, ok := a.helpers.Editing.Selected(a.ContextTree)
	if !ok {
		return a.c.Tr.SelectionInOneFile
	}
	words := presentation.LinesWords(a.c.Tr, at.Side, at.Line, at.StartLine, at.StartSide)
	if a.Where.Key == helpers.LinesKey(at) && a.Where.Words != "" {
		return words + " · " + a.Where.Words
	}
	return words
}

func (a App) homeMainBox(r views.Rect) []string {
	h := a.Home
	v := presentation.HomeMainView{List: h.List, Failed: h.Failed, Loading: h.Loading, Home: a.c.Home}
	v.Repo, v.Unit, v.Pull, v.Chosen = h.Pull()
	v.Row, _ = h.Selected()
	title, content := presentation.HomeMain(a.c.Tr, a.c.Theme, a.c.Markdown, a.controllers.Labels(context.HomeKey), r.Inner().W, v)
	return views.Framed(content, views.Extent{}, r.W, r.H, views.PanelTitle(context.MainPanel, title), "", "", false, a.c.Theme)
}
