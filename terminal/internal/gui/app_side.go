package gui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func panelNames(tr *i18n.TranslationSet) [context.SideCount + 1]string {
	return [context.SideCount + 1]string{tr.MainTitle, tr.ReviewTitle, tr.FilesTitle, tr.NotesTitle, tr.DiagramTitle}
}

func (a App) panelName() string {
	tr := a.c.Tr
	if a.Focus() != context.MainPanel {
		return panelNames(tr)[a.Focus()]
	}
	return map[context.Screen]string{context.ScreenReview: tr.ReviewTitle, context.ScreenDiff: tr.DiffTitle, context.ScreenDiagram: tr.DiagramTitle}[a.Screen()]
}

func (a App) focusOn(n int) (tea.Model, tea.Cmd) {
	if a.Review == nil {
		return a, nil
	}
	if n == context.MainPanel {
		a.ContextTree = a.Focused(context.MainPanel, a.Driving())
		return a.relaid(), nil
	}
	a.ContextTree = a.Focused(n, n)
	a = a.relaid()
	return a.followed()
}

func (a App) sidesBuilt() App {
	d, tr, th := *a.Review, a.c.Tr, a.c.Theme
	built := [context.SideCount + 1][]presentation.SideRow{
		context.ReviewPanel:  presentation.ReviewRows(th, d, a.Filter),
		context.FilesPanel:   presentation.FileRows(tr, th, d, a.FilesPanel.Folded, a.Filter),
		context.NotesPanel:   presentation.NoteListRows(tr, th, d, a.Filter),
		context.DiagramPanel: presentation.BoxRows(tr, th, d, a.Diagram.Level, a.Diagram.Grid, a.Filter),
	}
	for n := context.ReviewPanel; n <= context.SideCount; n++ {
		a.ContextTree = a.WithSide(n, a.Side(n).Reset(built[n]))
	}
	for n := context.ReviewPanel; n <= context.SideCount; n++ {
		a.ContextTree = a.WithSide(n, a.Side(n).Shown(a.sideHeight(n)))
	}
	return a
}

func (a App) mirrored() App {
	if a.Review == nil || a.Width == 0 {
		return a
	}
	pick := func(n int, key string) {
		if (n == a.Focus() && n != context.DiagramPanel) || key == "" {
			return
		}
		s := a.Side(n)
		if i := s.Index(key); i >= 0 {
			a.ContextTree = a.WithSide(n, s.To(i, a.sideHeight(n)))
		}
	}
	switch a.Screen() {
	case context.ScreenReview:
		if v := a.ReviewView.View; v.Cursor < len(v.Items) && v.Items[v.Cursor].Pull != 0 {
			pick(context.ReviewPanel, presentation.PullKey(v.Items[v.Cursor].Pull))
		}
	case context.ScreenDiff:
		if path := a.Diff.File(); path != "" {
			pick(context.FilesPanel, a.shownFile(path))
		}
		if v := a.Diff; v.Cursor < len(v.Rows) && (v.Rows[v.Cursor].Kind == "notehead" || v.Rows[v.Cursor].Kind == "notetext") {
			pick(context.NotesPanel, v.Rows[v.Cursor].Key)
		}
	case context.ScreenDiagram:
		a = a.sidesBuilt()
		if card, ok := a.Diagram.Card(); ok {
			pick(context.DiagramPanel, presentation.BoxKey(card.ID))
		}
	}
	return a
}

func (a App) shownFile(path string) string {
	if a.Side(context.FilesPanel).Index(presentation.FileKey(path)) >= 0 {
		return presentation.FileKey(path)
	}
	component := a.componentOf(path)
	if dir := presentation.DirKey(component, path); a.Side(context.FilesPanel).Index(dir) >= 0 {
		return dir
	}
	return presentation.ComponentKey(component)
}

func (a App) followed() (tea.Model, tea.Cmd) {
	row, ok := a.Side(a.Focus()).Selected()
	if !ok {
		return a, nil
	}
	switch a.Focus() {
	case context.ReviewPanel:
		a.ContextTree = a.Showing(context.ScreenReview)
		return a.grown(row).relaid(), nil
	case context.FilesPanel:
		place := row.Item.Place
		if row.Item.Kind == "component" {
			place = nil
		}
		next, cmd := a.openPlace(row.Item.Component, place)
		if place != nil && cmd == nil {
			after := next.(App)
			after.Diff = after.Diff.Find(*place).AtTop(after.mainHeight())
			return after.mirrored(), nil
		}
		return next.(App).mirrored(), cmd
	case context.NotesPanel:
		return a.toNote(row)
	case context.DiagramPanel:
		if i := a.Diagram.Level.Index(row.Item.Box.Box); i >= 0 {
			a.Diagram.Focus = i
		}
		a.ContextTree = a.Showing(context.ScreenDiagram)
		return a.diagramShown().mirrored(), nil
	}
	return a, nil
}

func (a App) toNote(row presentation.SideRow) (tea.Model, tea.Cmd) {
	place := row.Item.Place
	component := a.componentOf(place.Path)
	if component == "" {
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.NotInDiff, map[string]string{"path": place.Path}), true), nil
	}
	next, cmd := a.openPlace(component, place)
	if cmd != nil {
		return next, cmd
	}
	a = next.(App)
	a.Diff = a.Diff.Copied()
	a.Diff.Notes[row.Key] = true
	a.Diff = a.Diff.Rebuilt(a.c.Tr, a.md, a.opts.Location, *a.Review, a.Filter, a.mainHeight())
	if i := a.Diff.First(func(r presentation.DiffRow) bool { return r.Kind == "notehead" && r.Key == row.Key }); i >= 0 {
		a.Diff = a.Diff.To(i, a.mainHeight()).Placed(a.mainHeight())
	}
	return a.mirrored(), nil
}

func (a App) sideAction(action string, count int) (tea.Model, tea.Cmd) {
	s, h := a.Side(a.Focus()), a.sideHeight(a.Focus())
	row, selected := s.Selected()
	if a.Focus() == context.DiagramPanel {
		return a.boxAction(action, count)
	}
	switch action {
	case "down":
		a.ContextTree = a.WithSide(a.Focus(), s.Move(count, h))
		return a.followed()
	case "up":
		a.ContextTree = a.WithSide(a.Focus(), s.Move(-count, h))
		return a.followed()
	case "top":
		a.ContextTree = a.WithSide(a.Focus(), s.To(0, h))
		return a.followed()
	case "bottom":
		a.ContextTree = a.WithSide(a.Focus(), s.To(s.Landing(len(s.Rows)-1, -1), h))
		return a.followed()
	case "open":
		if !selected {
			return a, nil
		}
		if a.Focus() == context.FilesPanel && (row.Item.Kind == "component" || row.Item.Kind == "dir") {
			return a.treeFolded(row)
		}
		a = a.leave()
		a.ContextTree = a.Focused(context.MainPanel, a.Driving())
		return a.relaid(), nil
	case "next panel":
		return a.focusOn(context.ReviewPanel + utils.Around(a.Focus()-context.ReviewPanel+1, context.SideCount))
	case "previous panel":
		return a.focusOn(context.ReviewPanel + utils.Around(a.Focus()-context.ReviewPanel-1, context.SideCount))
	case "toggle fold":
		if selected && (row.Item.Kind == "component" || row.Item.Kind == "dir") {
			return a.treeFolded(row)
		}
	case "post":
		if selected {
			return a.openPosting(row.Item.Pull), nil
		}
	case "highlight":
		if selected {
			return a.highlight(row), nil
		}
	case "edit":
		return a.editQueued()
	case "drop":
		return a.dropQueued(), nil
	}
	return a, nil
}

func (a App) treeFolded(row presentation.SideRow) (tea.Model, tea.Cmd) {
	a.FilesPanel.Folded = utils.CopyBools(a.FilesPanel.Folded)
	a.FilesPanel.Folded[row.Key] = !a.FilesPanel.Folded[row.Key]
	return a.relaid(), nil
}

func (a App) boxAction(action string, count int) (tea.Model, tea.Cmd) {
	switch action {
	case "down":
		action = "next box"
	case "up":
		action = "previous box"
	}
	a.ContextTree = a.Showing(context.ScreenDiagram)
	next, cmd := a.diagramAction(action, count)
	after := next.(App)
	if after.Screen() != context.ScreenDiagram {
		after.ContextTree = after.Focused(context.MainPanel, after.Driving())
	}
	return after.relaid(), cmd
}
