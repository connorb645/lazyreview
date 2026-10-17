package gui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func (a App) barLine() string {
	th := a.c.Theme
	if a.Has(context.PromptKey) {
		return utils.Fit(" "+a.Prompt.Input.View(), a.Width)
	}
	right := ""
	status := a.helpers.Home.Status(a.ContextTree)
	switch {
	case a.Notice != "":
		style := th.Text
		if a.Bad {
			style = th.Bad
		}
		right = style.Render(ansi.Truncate(a.Notice, max(10, a.Width*2/3), "…"))
	case a.Keys.Typed() != "":
		right = th.Accent.Render(a.Keys.Typed())
	case status != "":
		right = th.Muted.Render(ansi.Truncate(status, max(10, a.Width*2/3), "…"))
	}
	hints := a.controllers.Hints(a.ContextTree)
	room := a.Width - ansi.StringWidth(right) - 3
	sep := th.Faint.Render(" | ")
	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = th.Muted.Render(h[1]+": ") + th.Accent.Render(h[0])
	}
	for len(parts) > 1 && ansi.StringWidth(strings.Join(parts, sep)) > room {
		parts = append(parts[:len(parts)-2], parts[len(parts)-1])
	}
	left := " " + ansi.Truncate(strings.Join(parts, sep), max(0, room), "…")
	if right != "" {
		right += " "
	}
	return utils.Spread(left, right, a.Width)
}
