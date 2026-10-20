package gui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/gui/views"
	"github.com/connorb645/lazyreview/internal/utils"
)

// View draws the screen: the panels on the left, the main panel on the
// right, any popup over them, and a bar of the keys that matter here. The
// home screen has a panel on the left for each repository; a review has its
// four.
func (a App) View() string {
	if a.Width == 0 || a.Height == 0 {
		return ""
	}
	window := a.helpers.Window
	rows := make([]string, max(0, a.Height-1))
	place := func(r views.Rect, lines []string) {
		for i, line := range lines {
			if y := r.Y + i; y >= 0 && y < len(rows) {
				rows[y] += line
			}
		}
	}
	if a.Screen() == context.ScreenHome {
		l := window.HomeLayout(a.ContextTree)
		for i, r := range l.Panels {
			if r.W > 2 && r.H > 0 {
				place(r, a.homePanelBox(i, r, a.Home.Focus == i && !a.Popped()))
			}
		}
		if r := l.Main; r.Shown() {
			place(r, a.homeMainBox(r))
		}
	} else {
		l := window.Layout(a.ContextTree)
		for n := context.ReviewPanel; n <= context.SideCount; n++ {
			if r := l.Sides[n]; r.W > 2 && r.H > 0 {
				place(r, a.sideBox(n, r, a.Focus() == n && !a.Popped()))
			}
		}
		if r := l.Main; r.Shown() {
			place(r, a.mainBox(r, a.Focus() == context.MainPanel && !a.Popped()))
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

func (a App) popup() []string {
	tr, th, window := a.c.Tr, a.c.Theme, a.helpers.Window
	switch {
	case a.Has(context.ConfirmationKey):
		labels, width := a.controllers.Labels(context.ConfirmationKey), min(a.Width-4, 56)
		if c := a.Confirmation.Conversation; c != nil {
			return presentation.DeleteBox(tr, th, labels, *c, width)
		}
		return presentation.DropBox(tr, th, labels, a.Confirmation.Comment, a.Confirmation.Pull, width)
	case a.Has(context.PostingKey):
		pn := a.Posting
		m, _ := a.helpers.Posting.QueueMember(a.ContextTree, pn.Pull)
		return presentation.PostingBox(tr, th, a.Review.State.Queue.Repo, pn.Pull, m, pn.Actions, pn.Choice, a.Has(context.PostingConfirmationKey), window.PopupWidth(a.ContextTree))
	case a.MenuOpen():
		return a.Menu.Box(tr, th, a.Width, min(a.Height-3, window.MenuRows(a.ContextTree)+2))
	}
	return nil
}
