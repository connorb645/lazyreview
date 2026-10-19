package gui

import (
	"fmt"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
)

func (a App) sideBox(n int, r views.Rect, focused bool) []string {
	s, tr := a.Side(n), a.c.Tr
	content, place := presentation.SideContent(tr, a.c.Theme, s.Rows, s.Cursor, s.Top, r.Inner(), focused)
	if n == context.ReviewPanel && a.GrowFrom != 0 {
		place = tr.Visual + " · " + place
	}
	return views.Framed(content, views.Extent{}, r.W, r.H, views.PanelTitle(n, presentation.PanelNames(tr)[n]), "", place, focused, a.c.Theme)
}

func (a App) homePanelBox(i int, r views.Rect, focused bool) []string {
	panel, th := a.Home.Panels[i], a.c.Theme
	content, place := presentation.SideContent(a.c.Tr, th, panel.Rows.Rows, panel.Rows.Cursor, panel.Rows.Top, r.Inner(), focused)
	if panel.Rows.Cursor < 0 {
		place = ""
	}
	title, waiting, tags := panel.Title, th.Warning, ""
	if panel.Hidden {
		title, waiting = a.c.Tr.HiddenTag+" "+title, th.Faint
	}
	if panel.Number > 0 {
		title = views.PanelTitle(panel.Number, title)
	}
	if panel.Waiting > 0 {
		tags = " " + waiting.Render(fmt.Sprintf("● %d", panel.Waiting))
	}
	title = ansi.Truncate(title, max(0, r.W-4-ansi.StringWidth(tags)), "…")
	if panel.Hidden && !focused {
		title = th.Faint.Render(title)
	}
	return views.Framed(content, views.Extent{}, r.W, r.H, title, tags, place, focused, th)
}
