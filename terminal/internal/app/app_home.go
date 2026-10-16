package app

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/server"
)

func readPulls(c *server.Client) tea.Cmd {
	return func() tea.Msg {
		pulls, err := c.Pulls()
		return pullsRead{pulls: pulls, err: err}
	}
}

func loadPulls(c *server.Client, unit string, links []string, account string) tea.Cmd {
	return func() tea.Msg {
		id, notes, err := c.Load(links, account)
		return pullsLoaded{unit: unit, id: id, notes: notes, err: err}
	}
}

func unitName(repo server.PullRepo, u server.Unit) string {
	numbers := make([]string, len(u.Members))
	for i, m := range u.Members {
		numbers[i] = fmt.Sprintf("#%d", m.Number)
	}
	return repo.Name + " " + strings.Join(numbers, " → ")
}

func (a App) homeAction(action string, count int) (tea.Model, tea.Cmd) {
	switch action {
	case "down":
		a.home = a.home.moved(count)
	case "up":
		a.home = a.home.moved(-count)
	case "top":
		a.home = a.home.ends(false)
	case "bottom":
		a.home = a.home.ends(true)
	case "open":
		return a.openSelected()
	default:
		if n, ok := strings.CutPrefix(action, "repository "); ok {
			number, _ := strconv.Atoi(n)
			a.home = a.home.toRepo(number)
		}
	}
	return a.homeRelaid(), nil
}

func (a App) goHome() (tea.Model, tea.Cmd) {
	if a.screen != screenHome {
		a.home.from, a.home.fromFocus, a.home.fromDriving = a.screen, a.focus, a.driving
	}
	a.screen, a.menu = screenHome, nil
	if a.home.list == nil && !a.home.reading && a.home.failed == "" {
		return a.readHome()
	}
	a.home = a.home.rebuilt(a.p)
	return a.homeRelaid(), nil
}

func (a App) readHome() (tea.Model, tea.Cmd) {
	if a.home.reading {
		return a.say("Reading GitHub already", false), nil
	}
	a.home.reading, a.home.failed = true, ""
	a.home = a.home.rebuilt(a.p)
	return a.homeRelaid(), readPulls(a.opts.Server)
}

func (a App) afterPulls(msg pullsRead) (tea.Model, tea.Cmd) {
	a.home.reading = false
	switch {
	case msg.err != nil && a.home.list != nil:
		a = a.say("Could not read GitHub again: "+msg.err.Error(), true)
	case msg.err != nil:
		a.home.failed = msg.err.Error()
	default:
		a.home.list = &msg.pulls
	}
	a.home = a.home.rebuilt(a.p)
	return a.homeRelaid(), nil
}

func (a App) openSelected() (tea.Model, tea.Cmd) {
	repo, u, _, ok := a.home.pull()
	switch {
	case !ok:
		return a, nil
	case u.Review != "":
		return a.openReview(u.Review)
	case a.home.loading != "":
		return a.say("One load at a time: "+a.home.loadingName+" is loading", true), nil
	case a.heldDraft() != "":
		return a.say(a.heldDraft(), true), nil
	}
	links := make([]string, len(u.Members))
	for i, m := range u.Members {
		links[i] = m.URL
	}
	a.home.loading, a.home.loadingName = unitKey(repo, u), unitName(repo, u)
	return a, loadPulls(a.opts.Server, a.home.loading, links, repo.Account)
}

func (a App) afterLoaded(msg pullsLoaded) (tea.Model, tea.Cmd) {
	name := a.home.loadingName
	a.home.loading, a.home.loadingName = "", ""
	if msg.err != nil {
		return a.say("Could not load "+name+": "+msg.err.Error(), true), nil
	}
	a.home = a.home.withReview(msg.unit, msg.id).rebuilt(a.p)
	if a.screen != screenHome {
		return a.say("Loaded "+name+"; Enter on the home screen opens it", false), nil
	}
	next, cmd := a.homeRelaid().openReview(msg.id)
	if len(msg.notes) > 0 {
		next = next.(App).say(strings.Join(msg.notes, " "), false)
	}
	return next, cmd
}

func (a App) openReview(id string) (tea.Model, tea.Cmd) {
	switch {
	case a.review != nil && a.review.id == id:
		a.screen, a.focus, a.driving = a.home.from, a.home.fromFocus, a.home.fromDriving
		if a.screen == screenHome {
			a.screen, a.focus, a.driving = screenReview, reviewPanel, reviewPanel
		}
		return a.relaid(), nil
	case a.loading && a.opening == id:
		a.screen, a.focus, a.driving = screenReview, reviewPanel, reviewPanel
		return a, nil
	case a.heldDraft() != "":
		return a.say(a.heldDraft(), true), nil
	}
	return a.cleared(id), loadReview(a.opts.Server, id)
}

func (a App) cleared(id string) App {
	if a.stopLive != nil {
		a.stopLive()
	}
	fresh := New(a.opts)
	fresh.md, fresh.width, fresh.height, fresh.mode, fresh.home = a.md, a.width, a.height, a.mode, a.home
	fresh.opening, fresh.loading, fresh.screen, fresh.focus, fresh.driving = id, true, screenReview, reviewPanel, reviewPanel
	return fresh
}

func (a App) heldDraft() string {
	if a.review == nil || (len(a.drafts) == 0 && !a.editingNow) {
		return ""
	}
	return "Your draft in " + a.review.state.Title + " is not sent yet: Space v goes back to it"
}

func (a App) homeLayout() homeLayout {
	return layHome(a.width, a.height, a.home.focus, a.mode, a.home.wants())
}

func (a App) homeRelaid() App {
	if len(a.home.panels) == 0 {
		a.home = a.home.rebuilt(a.p)
	}
	if a.width == 0 {
		return a
	}
	l := a.homeLayout()
	heights := make([]int, len(l.panels))
	for i, r := range l.panels {
		heights[i] = r.inner().h
	}
	a.home = a.home.shown(heights)
	return a
}

func (a App) homeMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return a, nil
	}
	for i, r := range a.homeLayout().panels {
		if r.w == 0 || !r.holds(msg.X, msg.Y) {
			continue
		}
		in := r.inner()
		switch msg.Button {
		case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
			by := 3
			if msg.Button == tea.MouseButtonWheelUp {
				by = -3
			}
			panels := append([]homePanel{}, a.home.panels...)
			panels[i].rows = panels[i].rows.scroll(by, max(1, in.h))
			a.home.panels = panels
			return a, nil
		case tea.MouseButtonLeft:
			now := a.opts.Now()
			double := msg.Y == a.clickedY && now.Sub(a.clickedAt) < doubleClick
			a.clickedAt, a.clickedY = now, msg.Y
			row := a.home.panels[i].rows.at(msg.Y - in.y)
			if row < 0 {
				a.home.focus = i
				return a.homeRelaid(), nil
			}
			a.home = a.home.at(i, row)
			if double {
				return a.homeRelaid().openSelected()
			}
			return a.homeRelaid(), nil
		}
	}
	return a, nil
}

func (a App) homeHints() [][2]string {
	if _, u, _, ok := a.home.pull(); ok {
		open := [2]string{"Enter", "Open"}
		if u.Review == "" {
			open = [2]string{"Enter", "Load and open"}
		}
		hints := [][2]string{open, {"gx", "GitHub"}, {"R", "Refresh"}}
		if numbers := a.home.numbers(3); len(numbers) > 1 {
			hints = append(hints, [2]string{strings.Join(numbers, " "), "Repositories"})
		}
		return append(hints, [2]string{"Space ?", "Keys"})
	}
	if a.home.list == nil && a.home.reading {
		return [][2]string{{":q", "Quit"}, {"Space ?", "Keys"}}
	}
	return [][2]string{{"R", "Read again"}, {":q", "Quit"}, {"Space ?", "Keys"}}
}

func (a App) homeStatus() string {
	switch {
	case a.home.loading != "":
		return "Loading " + a.home.loadingName + " from GitHub…"
	case a.home.reading && a.home.list != nil:
		return "Reading GitHub…"
	}
	return ""
}
