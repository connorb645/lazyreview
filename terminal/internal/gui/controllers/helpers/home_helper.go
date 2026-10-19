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
// loads one into a review, opens a review from it, and hides a repository
// from it or shows one again.
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
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.LoadedOpenFromHome, map[string]string{"name": name, "open": self.c.KeyLabels(context.HomeKey).Of("open")}), false), nil
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
	return utils.ResolvePlaceholderString(self.c.Tr.DraftNotSent, map[string]string{"title": s.Review.State.Title, "review": self.c.KeyLabels(s.Place()).Of("review")})
}

// HideOrShow is the state with the menu that hides the selected
// repository, or every repository of its owner, open. On a hidden one it is
// the command that shows it again or, when an entry for every repository of
// its owner hides it, the menu that says so and takes that entry off.
func (self *HomeHelper) HideOrShow(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	tr := self.c.Tr
	repo, ok := s.Home.Repo()
	if !ok {
		return s, nil
	}
	if len(repo.HiddenBy) == 0 {
		owner := strings.Split(repo.Name, "/")[0]
		pn := context.MenuContext{Menu: presentation.Menu{Title: tr.HideTitle, Rows: []presentation.MenuRow{
			{Left: repo.Name, Right: tr.HideThis, Open: &presentation.Item{Kind: "hide", Title: repo.Name}},
			{Left: owner + "/*", Right: utils.ResolvePlaceholderString(tr.HideOwner, map[string]string{"owner": owner}), Open: &presentation.Item{Kind: "hide", Title: owner + "/*"}},
		}}}
		return s.WithMenu(pn), nil
	}
	var owners []string
	for _, entry := range repo.HiddenBy {
		if strings.HasSuffix(entry, "/*") {
			owners = append(owners, entry)
		}
	}
	if len(owners) == 0 {
		return s, self.changeHidden(s, "", repo.HiddenBy)
	}
	entries := strings.Join(owners, ", ")
	pn := context.MenuContext{Menu: presentation.Menu{Title: utils.ResolvePlaceholderString(tr.HiddenByTitle, map[string]string{"entry": entries}), Rows: []presentation.MenuRow{
		{Left: entries, Right: utils.ResolvePlaceholderString(tr.ShowOwnerAgain, map[string]string{"owner": strings.TrimSuffix(owners[0], "/*")}), Open: &presentation.Item{Kind: "show"}},
	}}}
	return s.WithMenu(pn), nil
}

// Hide is the command that puts an entry on the config's hidden
// repositories: a repository, or every repository of an owner.
func (self *HomeHelper) Hide(s context.ContextTree, entry string) tea.Cmd {
	return self.changeHidden(s, entry, nil)
}

// Show is the command that takes every entry hiding the selected repository
// off the config's hidden repositories.
func (self *HomeHelper) Show(s context.ContextTree) tea.Cmd {
	repo, ok := s.Home.Repo()
	if !ok || len(repo.HiddenBy) == 0 {
		return nil
	}
	return self.changeHidden(s, "", repo.HiddenBy)
}

// AfterHidden is the state once the config's hidden repositories changed:
// the panels as they now hide, and a notice of what changed, or why it did
// not. With none hidden any more, the hidden ones are no longer shown.
func (self *HomeHelper) AfterHidden(s context.ContextTree, msg types.HiddenChanged) context.ContextTree {
	tr := self.c.Tr
	if msg.Err != nil {
		return s.Said(tr.CouldNotChangeHidden+": "+msg.Err.Error(), true)
	}
	s.Home = s.Home.WithHidden(msg.Hidden)
	if presentation.HiddenCount(s.Home.List) == 0 {
		s.Home.ShowHidden = false
	}
	s.Home = s.Home.Rebuilt(tr, self.c.Theme)
	var notice string
	switch {
	case strings.HasSuffix(msg.Hide, "/*"):
		notice = utils.ResolvePlaceholderString(tr.HidOwner, map[string]string{"owner": strings.TrimSuffix(msg.Hide, "/*")})
	case msg.Hide != "":
		notice = utils.ResolvePlaceholderString(tr.HidRepo, map[string]string{"repo": msg.Hide})
	case len(msg.Show) > 0:
		notice = utils.ResolvePlaceholderString(tr.ShowsAgain, map[string]string{"repo": msg.Show[0]})
	}
	for _, entry := range msg.Show {
		if strings.HasSuffix(entry, "/*") {
			notice = utils.ResolvePlaceholderString(tr.OwnerShowsAgain, map[string]string{"owner": strings.TrimSuffix(entry, "/*")})
		}
	}
	if show := self.c.KeyLabels(context.HomeKey).Of("show hidden"); msg.Hide != "" && !s.Home.ShowHidden && show != "" {
		shows := tr.ShowsIt
		if strings.HasSuffix(msg.Hide, "/*") {
			shows = tr.ShowsThem
		}
		notice += "; " + utils.ResolvePlaceholderString(shows, map[string]string{"key": show})
	}
	return self.c.Helpers.Refresh.HomeRelaid(s).Said(notice, false)
}

// ToggleHidden is the state with the repositories the config hides shown,
// dim, after the others, or no longer shown.
func (self *HomeHelper) ToggleHidden(s context.ContextTree) context.ContextTree {
	if !s.Home.ShowHidden && presentation.HiddenCount(s.Home.List) == 0 {
		return s.Said(self.c.Tr.NoneHidden, false)
	}
	s.Home.ShowHidden = !s.Home.ShowHidden
	s.Home = s.Home.Rebuilt(self.c.Tr, self.c.Theme)
	return self.c.Helpers.Refresh.HomeRelaid(s)
}

// Status is what the bar says of the home screen's reading and loading,
// or how many repositories the config hides from it.
func (self *HomeHelper) Status(s context.ContextTree) string {
	tr := self.c.Tr
	switch {
	case s.Home.Loading != "":
		return utils.ResolvePlaceholderString(tr.LoadingFromGitHub, map[string]string{"name": s.Home.LoadingName})
	case s.Home.Reading && s.Home.List != nil:
		return tr.ReadingGitHub
	case s.Screen() != context.ScreenHome:
		return ""
	}
	hidden := presentation.HiddenCount(s.Home.List)
	if hidden == 0 {
		return ""
	}
	words := [2][2]string{{tr.OneHidden, tr.ShowsIt}, {tr.ManyHidden, tr.ShowsThem}}
	if s.Home.ShowHidden {
		words = [2][2]string{{tr.ShowingOneHidden, tr.HidesIt}, {tr.ShowingManyHidden, tr.HidesThem}}
	}
	chosen := words[min(hidden, 2)-1]
	status := utils.ResolvePlaceholderString(chosen[0], map[string]string{"count": utils.Itoa(hidden)})
	if key := self.c.KeyLabels(context.HomeKey).Of("show hidden"); key != "" {
		status += " · " + utils.ResolvePlaceholderString(chosen[1], map[string]string{"key": key})
	}
	return status
}

func (self *HomeHelper) changeHidden(s context.ContextTree, hide string, show []string) tea.Cmd {
	c := self.c.Server
	names := make([]string, len(s.Home.List.Repos))
	for i, repo := range s.Home.List.Repos {
		names[i] = repo.Name
	}
	return func() tea.Msg {
		hidden, err := c.ChangeHidden(hide, show, names)
		return types.HiddenChanged{Hide: hide, Show: show, Hidden: hidden, Err: err}
	}
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
