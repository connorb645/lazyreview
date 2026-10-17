package controllers

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func reviewScreenBindings(c *helpers.HelperCommon) []Binding {
	tr, h := c.Tr, c.Helpers
	return []Binding{
		{Action: "next pull", Keys: []string{"] p"}, Handler: func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
			return h.Filter.FilterBy(s, count.N), nil
		}, Description: tr.NextPull},
		{Action: "previous pull", Keys: []string{"[ p"}, Handler: func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
			return h.Filter.FilterBy(s, -count.N), nil
		}, Description: tr.PreviousPull},
		{Action: "jump back", Keys: []string{"ctrl+o"}, Handler: func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
			return h.Jump.JumpBy(s, true)
		}, Description: tr.JumpBack},
		{Action: "jump forward", Keys: []string{"tab"}, Handler: func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
			return h.Jump.JumpBy(s, false)
		}, Description: tr.JumpForward},
		{Action: "diagram", Keys: []string{"space m"}, Handler: stateOnly(h.Diagram.OpenDiagram), Description: tr.Diagram},
		{Action: "finder", Keys: []string{"space f"}, Handler: stateOnly(h.Menu.OpenFinder), Description: tr.Find},
		{Action: "home", Keys: []string{"space h"}, Handler: uncounted(h.Home.GoHome), Description: tr.Home},
		{Action: "focus main", Keys: []string{"0"}, Handler: focusOn(h, context.MainPanel), Description: tr.FocusMain},
	}
}

func focusPanelBindings(c *helpers.HelperCommon) []Binding {
	names, h := presentation.PanelNames(c.Tr), c.Helpers
	return []Binding{
		{Action: "focus review", Keys: []string{"1"}, Handler: focusOn(h, context.ReviewPanel), Description: names[context.ReviewPanel]},
		{Action: "focus files", Keys: []string{"2"}, Handler: focusOn(h, context.FilesPanel), Description: names[context.FilesPanel]},
		{Action: "focus notes", Keys: []string{"3"}, Handler: focusOn(h, context.NotesPanel), Description: names[context.NotesPanel]},
		{Action: "focus diagram", Keys: []string{"4"}, Handler: focusOn(h, context.DiagramPanel), Description: names[context.DiagramPanel]},
	}
}

func focusOn(h *helpers.Helpers, n int) Handler {
	return func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
		return h.Panels.FocusOn(s, n)
	}
}

func reviewScreenHelp(c *helpers.HelperCommon, heading string, rows []presentation.HelpRow, labels presentation.KeyLabels) []HelpSection {
	tr := c.Tr
	command := map[string]string{"command": labels.Of("command")}
	every := []presentation.HelpRow{
		{Keys: labels.Of("focus review", "focus files", "focus notes", "focus diagram"), Words: tr.HelpFocusPanels},
		{Keys: labels.Of("focus main"), Words: tr.HelpFocusMain},
		{Keys: labels.Nth("close", 1), Words: tr.HelpEscape},
		{Keys: labels.Of("bigger", "smaller"), Words: utils.ResolvePlaceholderString(tr.HelpScreenMode, map[string]string{"smaller": labels.Of("smaller")})},
		{Keys: labels.Of("command"), Words: utils.ResolvePlaceholderString(tr.HelpCommandLine, command)},
		{Keys: labels.Of("keys"), Words: tr.HelpKeys},
		{Keys: refreshKeys(c, labels), Words: tr.HelpRefresh},
		{Keys: labels.Of("review"), Words: tr.HelpReviewView},
		{Keys: labels.Of("home"), Words: tr.HelpHome},
		{Keys: labels.Of("diagram"), Words: tr.HelpDiagram},
		{Keys: labels.Of("finder"), Words: tr.HelpFinder},
		{Keys: labels.Of("github"), Words: tr.HelpGitHub},
		{Keys: labels.Of("web"), Words: tr.HelpWeb},
		{Keys: labels.Of("next pull", "previous pull"), Words: tr.HelpPullFilter},
		{Keys: labels.Of("jump back", "jump forward"), Words: tr.HelpJumps},
		{Keys: labels.Of("close"), Words: tr.HelpClose},
		{Keys: labels.Of("quit"), Words: utils.ResolvePlaceholderString(tr.HelpQuit, command)},
	}
	return []HelpSection{{Heading: heading, Rows: rows}, {Heading: tr.EveryPanel, Rows: every}}
}

func reviewScreenHints(c *helpers.HelperCommon, s context.ContextTree, labels presentation.KeyLabels, hints func() [][2]string) [][2]string {
	tr := c.Tr
	if s.Review == nil {
		return [][2]string{{labels.Of("refresh"), tr.TryAgain}, {quitKeys(c, labels), tr.Quit}}
	}
	return append(hints(), [2]string{labels.Of("keys"), tr.Keys})
}

func refreshKeys(c *helpers.HelperCommon, labels presentation.KeyLabels) string {
	return utils.ResolvePlaceholderString(c.Tr.HelpRefreshKeys, map[string]string{"refresh": labels.Of("refresh"), "command": labels.Of("command")})
}

func quitKeys(c *helpers.HelperCommon, labels presentation.KeyLabels) string {
	return utils.ResolvePlaceholderString(c.Tr.QuitCommandKeys, map[string]string{"command": labels.Of("command")})
}
