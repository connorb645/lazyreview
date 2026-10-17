package gui

import (
	"slices"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func (a App) filterBy(by int) App {
	if a.Review == nil || a.Review.Page.Stack == nil {
		return a.say(a.c.Tr.OnlyStackFilters, true)
	}
	order := append([]int{0}, presentation.StackPulls(*a.Review)...)
	from := 0
	if len(a.Filter) > 0 {
		from = a.Filter[0]
		if by > 0 {
			from = a.Filter[len(a.Filter)-1]
		}
	}
	at := 0
	for i, pull := range order {
		if pull == from {
			at = i
		}
	}
	next := order[((at+by)%len(order)+len(order))%len(order)]
	if next == 0 {
		return a.setFilter(nil)
	}
	return a.setFilter(types.PullFilter{next})
}

func (a App) setFilter(f types.PullFilter) App {
	a.Filter, a.GrowFrom = f, 0
	return a.relaid()
}

func (a App) pullCommand(arg string) App {
	if a.Review == nil || a.Review.Page.Stack == nil {
		return a.say(a.c.Tr.OnlyStackFilters, true)
	}
	if arg == "all" {
		return a.setFilter(nil)
	}
	n, err := strconv.Atoi(strings.TrimPrefix(arg, "#"))
	if err != nil || presentation.StackMember(a.Review.Page.Stack, n).Head == "" {
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.NoPullInStack, map[string]string{"pull": arg}), true)
	}
	return a.setFilter(types.PullFilter{n})
}

func (a App) highlight(row presentation.SideRow) App {
	if a.Review.Page.Stack == nil || row.Item.Kind != "pull" {
		return a
	}
	a = a.setFilter(types.PullFilter{row.Item.Pull})
	a.GrowFrom = row.Item.Pull
	return a
}

func (a App) grown(row presentation.SideRow) App {
	if a.GrowFrom != 0 && row.Item.Kind == "pull" {
		a.Filter = pullRun(presentation.StackPulls(*a.Review), a.GrowFrom, row.Item.Pull)
	}
	return a
}

func (a App) growingInReview() App {
	if a.Focus() != context.ReviewPanel || a.Screen() == context.ScreenHome {
		a.GrowFrom = 0
	}
	return a
}

func (a App) keptFilter() App {
	stack := presentation.StackPulls(*a.Review)
	a.Filter = a.Filter.Within(stack)
	if !slices.Contains(stack, a.GrowFrom) {
		a.GrowFrom = 0
	}
	return a
}

func pullRun(stack []int, from, to int) types.PullFilter {
	in := false
	var run types.PullFilter
	for _, pull := range stack {
		edge := pull == from || pull == to
		if edge || in {
			run = append(run, pull)
		}
		if edge && (in || from == to) {
			break
		}
		if edge {
			in = true
		}
	}
	return run
}
