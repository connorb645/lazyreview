package gui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/utils"
)

func (a App) barLine() string {
	th := a.c.Theme
	if a.Has(context.PromptKey) {
		return utils.Fit(" "+a.Prompt.Input.View(), a.Width)
	}
	right := ""
	widest := max(10, a.Width*2/3)
	status := a.helpers.Home.Status(a.ContextTree)
	if status == "" {
		status = a.helpers.AppStatus.Status(a.ContextTree, widest)
	}
	switch {
	case a.Notice != "":
		style := th.Text
		if a.Bad {
			style = th.Bad
		}
		right = style.Render(ansi.Truncate(a.Notice, widest, "…"))
	case a.Keys.Typed() != "":
		right = th.Accent.Render(a.Keys.Typed())
	case status != "":
		right = th.Muted.Render(ansi.Truncate(status, widest, "…"))
	}
	hints := a.controllers.Hints(a.ContextTree)
	room := a.Width - 3
	if right != "" {
		room -= ansi.StringWidth(right) + 1
	}
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
