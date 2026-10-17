package gui

import (
	"fmt"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
)

func (a App) homePanelBox(i int, r views.Rect, focused bool) []string {
	panel, th := a.Home.Panels[i], a.c.Theme
	content, place := presentation.SideContent(a.c.Tr, th, panel.Rows.Rows, panel.Rows.Cursor, panel.Rows.Top, r.Inner(), focused)
	if panel.Rows.Cursor < 0 {
		place = ""
	}
	title := panel.Title
	if panel.Number > 0 {
		title = views.PanelTitle(panel.Number, title)
	}
	waiting := ""
	if panel.Waiting > 0 {
		waiting = " " + th.Warning.Render(fmt.Sprintf("● %d", panel.Waiting))
	}
	return views.Framed(content, views.Extent{}, r.W, r.H, title, waiting, place, focused, th)
}

func (a App) homeMainBox(r views.Rect) []string {
	h := a.Home
	v := presentation.HomeMainView{List: h.List, Failed: h.Failed, Loading: h.Loading, Home: a.opts.Home}
	v.Repo, v.Unit, v.Pull, v.Chosen = h.Pull()
	v.Row, _ = h.Selected()
	title, content := presentation.HomeMain(a.c.Tr, a.c.Theme, a.md, r.Inner().W, v)
	return views.Framed(content, views.Extent{}, r.W, r.H, views.PanelTitle(context.MainPanel, title), "", "", false, a.c.Theme)
}
