package gui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func readPulls(c *server.Client) tea.Cmd {
	return func() tea.Msg {
		pulls, err := c.Pulls()
		return types.PullsRead{Pulls: pulls, Err: err}
	}
}

func loadPulls(c *server.Client, unit string, links []string, account string) tea.Cmd {
	return func() tea.Msg {
		id, notes, err := c.Load(links, account)
		return types.PullsLoaded{Unit: unit, ID: id, Notes: notes, Err: err}
	}
}

func (a App) homeAction(action string, count int) (tea.Model, tea.Cmd) {
	switch action {
	case "down":
		a.Home = a.Home.Moved(count)
	case "up":
		a.Home = a.Home.Moved(-count)
	case "top":
		a.Home = a.Home.Ends(false)
	case "bottom":
		a.Home = a.Home.Ends(true)
	case "next panel":
		a.Home = a.Home.PanelBy(1)
	case "previous panel":
		a.Home = a.Home.PanelBy(-1)
	case "open":
		return a.openSelected()
	default:
		if n, ok := strings.CutPrefix(action, "repository "); ok {
			number, _ := strconv.Atoi(n)
			a.Home = a.Home.ToRepo(number)
		}
	}
	return a.homeRelaid(), nil
}

func (a App) goHome() (tea.Model, tea.Cmd) {
	if a.Screen() != context.ScreenHome {
		a.Home.From, a.Home.FromFocus, a.Home.FromDriving = a.Screen(), a.Focus(), a.Driving()
	}
	a.ContextTree = a.Showing(context.ScreenHome).Closed(context.MenuKey).Closed(context.FinderKey)
	if a.Home.List == nil && !a.Home.Reading && a.Home.Failed == "" {
		return a.readHome()
	}
	a.Home = a.Home.Rebuilt(a.c.Tr, a.c.Theme)
	return a.homeRelaid(), nil
}

func (a App) readHome() (tea.Model, tea.Cmd) {
	if a.Home.Reading {
		return a.say(a.c.Tr.ReadingAlready, false), nil
	}
	a.Home.Reading, a.Home.Failed = true, ""
	a.Home = a.Home.Rebuilt(a.c.Tr, a.c.Theme)
	return a.homeRelaid(), readPulls(a.opts.Server)
}

func (a App) afterPulls(msg types.PullsRead) (tea.Model, tea.Cmd) {
	a.Home.Reading = false
	switch {
	case msg.Err != nil && a.Home.List != nil:
		a = a.say(a.c.Tr.CouldNotReadGitHubAgain+": "+msg.Err.Error(), true)
	case msg.Err != nil:
		a.Home.Failed = msg.Err.Error()
	default:
		a.Home.List = &msg.Pulls
	}
	a.Home = a.Home.Rebuilt(a.c.Tr, a.c.Theme)
	return a.homeRelaid(), nil
}

func (a App) openSelected() (tea.Model, tea.Cmd) {
	repo, u, _, ok := a.Home.Pull()
	switch {
	case !ok:
		return a, nil
	case u.Review != "":
		return a.openReview(u.Review)
	case a.Home.Loading != "":
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.OneLoadAtATime, map[string]string{"name": a.Home.LoadingName}), true), nil
	case a.heldDraft() != "":
		return a.say(a.heldDraft(), true), nil
	}
	links := make([]string, len(u.Members))
	for i, m := range u.Members {
		links[i] = m.URL
	}
	a.Home.Loading, a.Home.LoadingName = presentation.UnitKey(repo, u), presentation.UnitName(repo, u)
	return a, loadPulls(a.opts.Server, a.Home.Loading, links, repo.Account)
}

func (a App) afterLoaded(msg types.PullsLoaded) (tea.Model, tea.Cmd) {
	name := a.Home.LoadingName
	a.Home.Loading, a.Home.LoadingName = "", ""
	if msg.Err != nil {
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.CouldNotLoad, map[string]string{"name": name})+": "+msg.Err.Error(), true), nil
	}
	a.Home = a.Home.WithReview(msg.Unit, msg.ID).Rebuilt(a.c.Tr, a.c.Theme)
	if a.Screen() != context.ScreenHome {
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.LoadedEnterOpens, map[string]string{"name": name}), false), nil
	}
	next, cmd := a.homeRelaid().openReview(msg.ID)
	if len(msg.Notes) > 0 {
		next = next.(App).say(strings.Join(msg.Notes, " "), false)
	}
	return next, cmd
}

func (a App) openReview(id string) (tea.Model, tea.Cmd) {
	switch {
	case a.Review != nil && a.Review.ID == id:
		a.ContextTree = a.Showing(a.Home.From).Focused(a.Home.FromFocus, a.Home.FromDriving)
		if a.Screen() == context.ScreenHome {
			a.ContextTree = a.Showing(context.ScreenReview).Focused(context.ReviewPanel, context.ReviewPanel)
		}
		return a.relaid(), nil
	case a.Loading && a.Opening == id:
		a.ContextTree = a.Showing(context.ScreenReview).Focused(context.ReviewPanel, context.ReviewPanel)
		return a, nil
	case a.heldDraft() != "":
		return a.say(a.heldDraft(), true), nil
	}
	return a.cleared(id), loadReview(a.c.Tr, a.opts.Server, id)
}

func (a App) cleared(id string) App {
	if a.StopLive != nil {
		a.StopLive()
	}
	fresh := New(a.c, a.opts)
	fresh.md, fresh.Width, fresh.Height, fresh.Mode, fresh.Home = a.md, a.Width, a.Height, a.Mode, a.Home
	fresh.Opening, fresh.Loading = id, true
	fresh.ContextTree = fresh.Showing(context.ScreenReview).Focused(context.ReviewPanel, context.ReviewPanel)
	return fresh
}

func (a App) heldDraft() string {
	if a.Review == nil || (len(a.Drafts) == 0 && !a.EditingNow) {
		return ""
	}
	return utils.ResolvePlaceholderString(a.c.Tr.DraftNotSent, map[string]string{"title": a.Review.State.Title})
}

func (a App) homeLayout() homeLayout {
	return layHome(a.Width, a.Height, a.Home.Focus, a.Mode, a.Home.Wants())
}

func (a App) homeRelaid() App {
	if len(a.Home.Panels) == 0 {
		a.Home = a.Home.Rebuilt(a.c.Tr, a.c.Theme)
	}
	if a.Width == 0 {
		return a
	}
	l := a.homeLayout()
	heights := make([]int, len(l.panels))
	for i, r := range l.panels {
		heights[i] = r.Inner().H
	}
	a.Home = a.Home.Shown(heights)
	return a
}

func (a App) homeMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return a, nil
	}
	for i, r := range a.homeLayout().panels {
		if r.W == 0 || !r.Holds(msg.X, msg.Y) {
			continue
		}
		in := r.Inner()
		switch msg.Button {
		case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
			by := 3
			if msg.Button == tea.MouseButtonWheelUp {
				by = -3
			}
			panels := append([]context.HomePanel{}, a.Home.Panels...)
			panels[i].Rows = panels[i].Rows.Scroll(by, max(1, in.H))
			a.Home.Panels = panels
			return a, nil
		case tea.MouseButtonLeft:
			now := a.opts.Now()
			double := msg.Y == a.ClickedY && now.Sub(a.ClickedAt) < doubleClick
			a.ClickedAt, a.ClickedY = now, msg.Y
			row := a.Home.Panels[i].Rows.At(msg.Y - in.Y)
			if row < 0 {
				a.Home.Focus = i
				return a.homeRelaid(), nil
			}
			a.Home = a.Home.At(i, row)
			if double {
				return a.homeRelaid().openSelected()
			}
			return a.homeRelaid(), nil
		}
	}
	return a, nil
}

func (a App) homeHints() [][2]string {
	tr := a.c.Tr
	if _, u, _, ok := a.Home.Pull(); ok {
		open := [2]string{"Enter", tr.Open}
		if u.Review == "" {
			open = [2]string{"Enter", tr.LoadAndOpen}
		}
		hints := [][2]string{open}
		if len(a.Home.Panels) > 1 {
			hints = append(hints, [2]string{"h l", tr.Repositories})
		}
		return append(hints, [2]string{"gx", tr.GitHub}, [2]string{"R", tr.Refresh}, [2]string{"Space ?", tr.Keys})
	}
	if a.Home.List == nil && a.Home.Reading {
		return [][2]string{{":q", tr.Quit}, {"Space ?", tr.Keys}}
	}
	return [][2]string{{"R", tr.ReadAgain}, {":q", tr.Quit}, {"Space ?", tr.Keys}}
}

func (a App) homeStatus() string {
	switch {
	case a.Home.Loading != "":
		return utils.ResolvePlaceholderString(a.c.Tr.LoadingFromGitHub, map[string]string{"name": a.Home.LoadingName})
	case a.Home.Reading && a.Home.List != nil:
		return a.c.Tr.ReadingGitHub
	}
	return ""
}
