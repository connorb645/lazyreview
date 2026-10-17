package presentation

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
)

// Tabs are a stack's tabs after the Review panel's title: All and each pull
// request, those the filter shows lit, in the room there is. With no room
// for every tab, only the lit ones show, then the first and last of them.
func Tabs(tr *i18n.TranslationSet, th theme.Theme, d types.ReviewData, filter types.PullFilter, room int) string {
	tab := func(name string, on bool) string {
		if on {
			return th.Accent.Bold(true).Render(name)
		}
		return th.Muted.Render(name)
	}
	names, on := []string{tr.All}, []bool{len(filter) == 0}
	for _, m := range d.State.Members {
		names = append(names, fmt.Sprintf("#%d", m.Pull))
		on = append(on, slices.Contains(filter, m.Pull))
	}
	var all, active []string
	for i, name := range names {
		all = append(all, tab(name, on[i]))
		if on[i] {
			active = append(active, tab(name, true))
		}
	}
	sep := th.InactiveBorder.Render(" - ")
	fits := []string{sep + strings.Join(all, sep), sep + strings.Join(active, sep)}
	if th.Plain {
		fits = fits[1:]
	}
	if len(active) > 2 {
		fits = append(fits, sep+active[0]+th.Muted.Render("-")+active[len(active)-1])
	}
	for _, out := range fits[:len(fits)-1] {
		if ansi.StringWidth(out) <= room {
			return out
		}
	}
	return fits[len(fits)-1]
}
