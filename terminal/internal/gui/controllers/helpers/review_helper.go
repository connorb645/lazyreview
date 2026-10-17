package helpers

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// ReviewHelper loads a review from the server, and loads it again when it
// changes.
type ReviewHelper struct {
	c *HelperCommon
}

// Load is the command that loads a review's state and page.
func (self *ReviewHelper) Load(id string) tea.Cmd {
	c := self.c.Server
	couldNotOpen := utils.ResolvePlaceholderString(self.c.Tr.CouldNotOpen, map[string]string{"name": id})
	return func() tea.Msg {
		state, err := c.State(id)
		if err != nil {
			return types.ReviewLoaded{ID: id, Err: fmt.Errorf("%s: %w", couldNotOpen, err)}
		}
		page, err := c.Page(id)
		if err != nil {
			return types.ReviewLoaded{ID: id, Err: fmt.Errorf("%s: %w", couldNotOpen, err)}
		}
		return types.ReviewLoaded{ID: id, Data: types.ReviewData{ID: id, State: state, Page: page}}
	}
}

// Reload is the command that loads the review open again, after the
// server refreshes it from GitHub when refresh is set.
func (self *ReviewHelper) Reload(id string, refresh bool) tea.Cmd {
	c, load := self.c.Server, self.Load(id)
	return func() tea.Msg {
		var notes []string
		if refresh {
			var err error
			if notes, err = c.Refresh(id); err != nil {
				return types.ReviewReloaded{Err: err}
			}
		}
		loaded, ok := load().(types.ReviewLoaded)
		if !ok || loaded.Err != nil {
			return types.ReviewReloaded{Err: loaded.Err}
		}
		return types.ReviewReloaded{Data: loaded.Data, Notes: notes}
	}
}

// LoadState is the command that loads the review's state again.
func (self *ReviewHelper) LoadState(id string) tea.Cmd {
	c := self.c.Server
	return func() tea.Msg {
		state, err := c.State(id)
		return types.StateLoaded{State: state, Err: err}
	}
}

// Refresh is the state loading the review again, from GitHub, or the home
// screen's list, and the command that does it.
func (self *ReviewHelper) Refresh(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	if s.Screen() == context.ScreenHome {
		return self.c.Helpers.Home.ReadHome(s)
	}
	if s.Review == nil {
		s.Loading, s.Failed = true, ""
		return s.Said(self.c.Tr.OpeningReviewAgain, false), self.Load(s.Opening)
	}
	cmd := self.Reload(s.Review.ID, true)
	if s.LiveFor == "" {
		var listening tea.Cmd
		s, listening = self.c.Helpers.Live.ListenTo(s, s.Review.ID)
		cmd = tea.Batch(cmd, listening)
	}
	return s.Said(self.c.Tr.LoadingReviewAgain, false), cmd
}

// Show is the state with the Review view in the main panel, the keys
// there, or a notice that no review is open.
func (self *ReviewHelper) Show(s context.ContextTree) context.ContextTree {
	switch {
	case s.Review == nil && s.Screen() == context.ScreenHome:
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.NoReviewOpenYet, map[string]string{"open": self.c.KeyLabels(context.HomeKey).Of("open")}), true)
	case s.Review == nil:
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.ReviewNotOpenYet, map[string]string{"refresh": self.c.KeyLabels(s.Place()).Of("refresh")}), true)
	case s.Screen() != context.ScreenHome:
		s = self.c.Helpers.Jump.Leave(s)
	}
	s = s.Showing(context.ScreenReview).Focused(context.MainPanel, context.ReviewPanel)
	return self.c.Helpers.Refresh.Relaid(s)
}

// AfterLoaded is the state once the review opening is loaded, everything
// of the one before it let go, or why it could not be.
func (self *ReviewHelper) AfterLoaded(s context.ContextTree, msg types.ReviewLoaded) (context.ContextTree, tea.Cmd) {
	if msg.ID != s.Opening {
		return s, nil
	}
	s.Loading = false
	if msg.Err != nil {
		s.Failed = msg.Err.Error()
		return s.Said(msg.Err.Error(), true), nil
	}
	s.Review, s.Failed = &msg.Data, ""
	if s.Screen() != context.ScreenHome {
		s = s.Showing(context.ScreenReview)
	}
	s.Files, s.Diff, s.Filter, s.GrowFrom = map[string]server.File{}, context.DiffContext{}, nil, 0
	s.Diagram = context.NewDiagramContext(self.c.Tr, context.LevelAt(self.c.Tr, msg.Data.Page, types.DiagramPlace{Level: "components"}), "")
	s = s.Closed(context.LabelsKey)
	var listening tea.Cmd
	s, listening = self.c.Helpers.Live.ListenTo(s, msg.Data.ID)
	return self.c.Helpers.Refresh.Relaid(s), listening
}

// AfterReload is the state with the review loaded again, the main panel
// keeping its place, or why it could not be.
func (self *ReviewHelper) AfterReload(s context.ContextTree, msg types.ReviewReloaded) (context.ContextTree, tea.Cmd) {
	if msg.Err != nil {
		return s.Said(msg.Err.Error(), true), nil
	}
	if s.Review == nil || msg.Data.ID != s.Review.ID {
		return s, nil
	}
	words := self.c.Tr.LoadedAgain
	if len(msg.Notes) > 0 {
		words += " " + strings.Join(msg.Notes, " ")
	}
	h := self.c.Helpers
	s.Review = &msg.Data
	s.Files = map[string]server.File{}
	s = h.Filter.KeptFilter(s)
	switch s.Screen() {
	case context.ScreenDiff:
		next, cmd := h.Diff.OpenPlace(s, s.Diff.Component, s.Diff.Place())
		return next.Said(words, false), cmd
	case context.ScreenDiagram:
		return h.Diagram.ShowBox(s, s.Diagram.Place()).Said(words, false), nil
	}
	return h.Refresh.Relaid(s).Said(words, false), nil
}

// AfterState is the state with the review's state loaded again, or why it
// could not be.
func (self *ReviewHelper) AfterState(s context.ContextTree, msg types.StateLoaded) context.ContextTree {
	if msg.Err != nil {
		return s.Said(self.c.Tr.CouldNotReadReviewAgain+": "+msg.Err.Error(), true)
	}
	return self.c.Helpers.Refresh.WithState(s, msg.State)
}
