package helpers

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// HomeHelper reads the open pull requests from GitHub for the home screen,
// loads one into a review, and opens a review from it.
type HomeHelper struct {
	c *HelperCommon
}

// ReadPulls is the command that reads the open pull requests.
func (self *HomeHelper) ReadPulls() tea.Cmd {
	c := self.c.Server
	return func() tea.Msg {
		pulls, err := c.Pulls()
		return types.PullsRead{Pulls: pulls, Err: err}
	}
}

// GoHome is the state with the home screen shown, keeping the review's
// place to come back to, and its list read if it has not been.
func (self *HomeHelper) GoHome(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	if s.Screen() != context.ScreenHome {
		s.Home.From, s.Home.FromFocus, s.Home.FromDriving = s.Screen(), s.Focus(), s.Driving()
	}
	s = s.Showing(context.ScreenHome).WithoutMenu()
	if s.Home.List == nil && !s.Home.Reading && s.Home.Failed == "" {
		return self.ReadHome(s)
	}
	s.Home = s.Home.Rebuilt(self.c.Tr, self.c.Theme)
	return self.c.Helpers.Refresh.HomeRelaid(s), nil
}

// ReadHome is the state reading the open pull requests again, and the
// command that reads them.
func (self *HomeHelper) ReadHome(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	if s.Home.Reading {
		return s.Said(self.c.Tr.ReadingAlready, false), nil
	}
	s.Home.Reading, s.Home.Failed = true, ""
	s.Home = s.Home.Rebuilt(self.c.Tr, self.c.Theme)
	return self.c.Helpers.Refresh.HomeRelaid(s), self.ReadPulls()
}

// AfterPulls is the state once the open pull requests are read, or why
// they could not be.
func (self *HomeHelper) AfterPulls(s context.ContextTree, msg types.PullsRead) (context.ContextTree, tea.Cmd) {
	s.Home.Reading = false
	switch {
	case msg.Err != nil && s.Home.List != nil:
		s = s.Said(self.c.Tr.CouldNotReadGitHubAgain+": "+msg.Err.Error(), true)
	case msg.Err != nil:
		s.Home.Failed = msg.Err.Error()
	default:
		s.Home.List = &msg.Pulls
	}
	s.Home = s.Home.Rebuilt(self.c.Tr, self.c.Theme)
	return self.c.Helpers.Refresh.HomeRelaid(s), nil
}

// OpenSelected is the state with the selected pull request's review open,
// or the command that loads it from GitHub first.
func (self *HomeHelper) OpenSelected(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	repo, u, _, ok := s.Home.Pull()
	switch {
	case !ok:
		return s, nil
	case u.Review != "":
		return self.OpenReview(s, u.Review)
	case s.Home.Loading != "":
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.OneLoadAtATime, map[string]string{"name": s.Home.LoadingName}), true), nil
	case self.HeldDraft(s) != "":
		return s.Said(self.HeldDraft(s), true), nil
	}
	links := make([]string, len(u.Members))
	for i, m := range u.Members {
		links[i] = m.URL
	}
	s.Home.Loading, s.Home.LoadingName = presentation.UnitKey(repo, u), presentation.UnitName(repo, u)
	return s, self.loadPulls(s.Home.Loading, links, repo.Account)
}

// AfterLoaded is the state once pull requests are loaded into a review:
// the review open, or a notice of it, or why the load failed.
func (self *HomeHelper) AfterLoaded(s context.ContextTree, msg types.PullsLoaded) (context.ContextTree, tea.Cmd) {
	name := s.Home.LoadingName
	s.Home.Loading, s.Home.LoadingName = "", ""
	if msg.Err != nil {
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.CouldNotLoad, map[string]string{"name": name})+": "+msg.Err.Error(), true), nil
	}
	s.Home = s.Home.WithReview(msg.Unit, msg.ID).Rebuilt(self.c.Tr, self.c.Theme)
	if s.Screen() != context.ScreenHome {
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.LoadedEnterOpens, map[string]string{"name": name}), false), nil
	}
	next, cmd := self.OpenReview(self.c.Helpers.Refresh.HomeRelaid(s), msg.ID)
	if len(msg.Notes) > 0 {
		next = next.Said(strings.Join(msg.Notes, " "), false)
	}
	return next, cmd
}

// OpenReview is the state back on a review: the one open, as it was left,
// or a new one opening, unless a draft is held in the one open.
func (self *HomeHelper) OpenReview(s context.ContextTree, id string) (context.ContextTree, tea.Cmd) {
	switch {
	case s.Review != nil && s.Review.ID == id:
		s = s.Showing(s.Home.From).Focused(s.Home.FromFocus, s.Home.FromDriving)
		if s.Screen() == context.ScreenHome {
			s = s.Showing(context.ScreenReview).Focused(context.ReviewPanel, context.ReviewPanel)
		}
		return self.c.Helpers.Refresh.Relaid(s), nil
	case s.Loading && s.Opening == id:
		s = s.Showing(context.ScreenReview).Focused(context.ReviewPanel, context.ReviewPanel)
		return s, nil
	case self.HeldDraft(s) != "":
		return s.Said(self.HeldDraft(s), true), nil
	}
	return cleared(s, id), self.c.Helpers.Review.Load(id)
}

// HeldDraft is why another review cannot open while a draft of the one
// open is not sent yet, or nothing.
func (self *HomeHelper) HeldDraft(s context.ContextTree) string {
	if s.Review == nil || (len(s.Drafts) == 0 && !s.EditingNow) {
		return ""
	}
	return utils.ResolvePlaceholderString(self.c.Tr.DraftNotSent, map[string]string{"title": s.Review.State.Title})
}

// Status is what the bar says of the home screen's reading and loading.
func (self *HomeHelper) Status(s context.ContextTree) string {
	switch {
	case s.Home.Loading != "":
		return utils.ResolvePlaceholderString(self.c.Tr.LoadingFromGitHub, map[string]string{"name": s.Home.LoadingName})
	case s.Home.Reading && s.Home.List != nil:
		return self.c.Tr.ReadingGitHub
	}
	return ""
}

func (self *HomeHelper) loadPulls(unit string, links []string, account string) tea.Cmd {
	c := self.c.Server
	return func() tea.Msg {
		id, notes, err := c.Load(links, account)
		return types.PullsLoaded{Unit: unit, ID: id, Notes: notes, Err: err}
	}
}

func cleared(s context.ContextTree, id string) context.ContextTree {
	if s.StopLive != nil {
		s.StopLive()
	}
	fresh := context.NewContextTree("")
	fresh.Width, fresh.Height, fresh.Mode, fresh.Home = s.Width, s.Height, s.Mode, s.Home
	fresh.Opening, fresh.Loading = id, true
	return fresh.Showing(context.ScreenReview).Focused(context.ReviewPanel, context.ReviewPanel)
}
