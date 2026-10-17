package gui

import (
	"fmt"

	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
)

func (a App) homePanelBox(i int, r views.Rect, focused bool) []string {
	panel, th := a.home.panels[i], a.c.Theme
	content, place := presentation.SideContent(a.c.Tr, th, panel.rows.rows, panel.rows.cursor, panel.rows.top, r.Inner(), focused)
	if panel.rows.cursor < 0 {
		place = ""
	}
	title := panel.title
	if panel.number > 0 {
		title = views.PanelTitle(panel.number, title)
	}
	waiting := ""
	if panel.waiting > 0 {
		waiting = " " + th.Warning.Render(fmt.Sprintf("● %d", panel.waiting))
	}
	return views.Framed(content, views.Extent{}, r.W, r.H, title, waiting, place, focused, th)
}

func (a App) homeMainBox(r views.Rect) []string {
	h := a.home
	v := presentation.HomeMainView{List: h.list, Failed: h.failed, Loading: h.loading, Home: a.opts.Home}
	v.Repo, v.Unit, v.Pull, v.Chosen = h.pull()
	v.Row, _ = h.selected()
	title, content := presentation.HomeMain(a.c.Tr, a.c.Theme, a.md, r.Inner().W, v)
	return views.Framed(content, views.Extent{}, r.W, r.H, views.PanelTitle(mainPanel, title), "", "", false, a.c.Theme)
}
