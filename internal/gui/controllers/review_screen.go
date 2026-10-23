package controllers

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/utils"
)

func reviewScreenBindings(c *helpers.HelperCommon) []Binding {
	tr, h, kb := c.Tr, c.Helpers, c.Config.Keybinding
	return []Binding{
		{Action: "next pull", Keys: kb.ReviewScreen.NextPull, Handler: func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
			return h.Filter.FilterBy(s, count.N), nil
		}, Description: tr.NextPull},
		{Action: "previous pull", Keys: kb.ReviewScreen.PreviousPull, Handler: func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
			return h.Filter.FilterBy(s, -count.N), nil
		}, Description: tr.PreviousPull},
		{Action: "jump back", Keys: kb.ReviewScreen.JumpBack, Handler: func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
			return h.Jump.JumpBy(s, -count.N)
		}, Description: tr.JumpBack},
		{Action: "jump forward", Keys: kb.ReviewScreen.JumpForward, Handler: func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
			return h.Jump.JumpBy(s, count.N)
		}, Description: tr.JumpForward},
		{Action: "diagram", Keys: kb.ReviewScreen.Diagram, Handler: stateOnly(h.Diagram.OpenDiagram), Description: tr.Diagram},
		{Action: "finder", Keys: kb.ReviewScreen.Finder, Handler: stateOnly(h.Menu.OpenFinder), Description: tr.Find},
		{Action: "home", Keys: kb.ReviewScreen.Home, Handler: uncounted(h.Home.GoHome), Description: tr.Home},
		{Action: "focus main", Keys: kb.ReviewScreen.FocusMain, Handler: focusOn(h, context.MainPanel), Description: tr.FocusMain},
	}
}

func focusPanelBindings(c *helpers.HelperCommon) []Binding {
	names, h, block := presentation.PanelNames(c.Tr), c.Helpers, c.Config.Keybinding.Panels.JumpToBlock
	return []Binding{
		{Action: "focus review", Keys: numbered(block, 0), Handler: focusOn(h, context.ReviewPanel), Description: names[context.ReviewPanel]},
		{Action: "focus files", Keys: numbered(block, 1), Handler: focusOn(h, context.FilesPanel), Description: names[context.FilesPanel]},
		{Action: "focus notes", Keys: numbered(block, 2), Handler: focusOn(h, context.NotesPanel), Description: names[context.NotesPanel]},
		{Action: "focus diagram", Keys: numbered(block, 3), Handler: focusOn(h, context.DiagramPanel), Description: names[context.DiagramPanel]},
	}
}

func focusOn(h *helpers.Helpers, n int) Handler {
	return func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
		return h.Panels.FocusOn(s, n)
	}
}

func reviewScreenHelp(c *helpers.HelperCommon, s context.ContextTree, heading string, rows []presentation.HelpRow, labels presentation.KeyLabels) []HelpSection {
	tr := c.Tr
	command := map[string]string{"command": labels.Of("command")}
	every := slices.Concat([]presentation.HelpRow{
		{Keys: labels.Of("focus review", "focus files", "focus notes", "focus diagram"), Words: tr.HelpFocusPanels},
		{Keys: labels.Of("focus main"), Words: tr.HelpFocusMain},
		{Keys: tr.CountKeys, Words: tr.HelpCount},
		{Keys: labels.Nth("close", 1), Words: tr.HelpEscape},
		{Keys: labels.Of("bigger", "smaller"), Words: utils.ResolvePlaceholderString(tr.HelpScreenMode, map[string]string{"smaller": labels.Of("smaller")})},
		{Keys: labels.Of("command"), Words: utils.ResolvePlaceholderString(tr.HelpCommandLine, command)},
		{Keys: labels.Of("keys"), Words: tr.HelpKeys},
		{Keys: refreshKeys(c, labels), Words: tr.HelpRefresh},
		{Keys: labels.Of("review"), Words: tr.HelpReviewView},
		{Keys: labels.Of("home"), Words: tr.HelpHome},
		{Keys: labels.Of("diagram"), Words: tr.HelpDiagram},
		{Keys: labels.Of("finder"), Words: tr.HelpFinder},
	}, only(onGitHub(s), presentation.HelpRow{Keys: labels.Of("github"), Words: tr.HelpGitHub}),
		only(filterable(s), presentation.HelpRow{Keys: labels.Of("next pull", "previous pull"), Words: tr.HelpPullFilter}), []presentation.HelpRow{
			{Keys: labels.Of("jump back", "jump forward"), Words: tr.HelpJumps},
			{Keys: labels.Of("close"), Words: tr.HelpClose},
			{Keys: labels.Of("quit"), Words: utils.ResolvePlaceholderString(tr.HelpQuit, command)},
		})
	return []HelpSection{{Heading: heading, Rows: rows}, {Heading: tr.EveryPanel, Rows: every}}
}

func reviewScreenHints(c *helpers.HelperCommon, s context.ContextTree, labels presentation.KeyLabels, hints func() [][2]string) [][2]string {
	tr := c.Tr
	if s.Review == nil {
		return [][2]string{{labels.Of("refresh"), tr.TryAgain}, {quitKeys(c, labels), tr.Quit}}
	}
	return append(hints(), [2]string{labels.Of("keys"), tr.Keys})
}

func takesComments(s context.ContextTree) bool {
	return s.Review != nil && s.Review.State.Queue != nil
}

func filterable(s context.ContextTree) bool {
	return takesComments(s) || s.Review != nil && s.Review.Page.Stack != nil
}

func only(act bool, rows ...presentation.HelpRow) []presentation.HelpRow {
	if !act {
		return nil
	}
	return rows
}

func forReview(s context.ContextTree, takingComments, readOnly string) string {
	if takesComments(s) {
		return takingComments
	}
	return readOnly
}

func dropWords(c *helpers.HelperCommon, s context.ContextTree, labels presentation.KeyLabels) string {
	return utils.ResolvePlaceholderString(forReview(s, c.Tr.HelpDropAtCursor, c.Tr.HelpDeleteAtCursor), map[string]string{"confirm": labels.Of("confirm")})
}

func canPost(s context.ContextTree) bool {
	return s.Review != nil && s.Review.State.Queue != nil && s.Review.State.Queue.Moved == nil
}

func commentable(s context.ContextTree, labels presentation.KeyLabels, hints [][2]string) [][2]string {
	if canPost(s) {
		return hints
	}
	comment := labels.Of("comment")
	return slices.DeleteFunc(hints, func(h [2]string) bool { return h[0] == comment })
}

func onGitHub(s context.ContextTree) bool {
	return s.Review != nil && s.Review.State.Kind != "branch" && len(s.Review.State.Members) > 0
}

func refreshKeys(c *helpers.HelperCommon, labels presentation.KeyLabels) string {
	return utils.ResolvePlaceholderString(c.Tr.HelpRefreshKeys, map[string]string{"refresh": labels.Of("refresh"), "command": labels.Of("command")})
}

func quitKeys(c *helpers.HelperCommon, labels presentation.KeyLabels) string {
	return utils.ResolvePlaceholderString(c.Tr.QuitCommandKeys, map[string]string{"command": labels.Of("command")})
}
