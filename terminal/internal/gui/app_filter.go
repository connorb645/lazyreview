package gui

import (
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func (a App) filterBy(by int) App {
	if a.review == nil || a.review.page.Stack == nil {
		return a.say(a.c.Tr.OnlyStackFilters, true)
	}
	order := append([]int{0}, stackPulls(*a.review)...)
	from := 0
	if len(a.filter) > 0 {
		from = a.filter[0]
		if by > 0 {
			from = a.filter[len(a.filter)-1]
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
	return a.setFilter(pullFilter{next})
}

func (a App) setFilter(f pullFilter) App {
	a.filter, a.growFrom = f, 0
	return a.relaid()
}

func (a App) pullCommand(arg string) App {
	if a.review == nil || a.review.page.Stack == nil {
		return a.say(a.c.Tr.OnlyStackFilters, true)
	}
	if arg == "all" {
		return a.setFilter(nil)
	}
	n, err := strconv.Atoi(strings.TrimPrefix(arg, "#"))
	if err != nil || stackMember(a.review.page.Stack, n).Head == "" {
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.NoPullInStack, map[string]string{"pull": arg}), true)
	}
	return a.setFilter(pullFilter{n})
}

func (a App) highlight(row sideRow) App {
	if a.review.page.Stack == nil || row.it.kind != "pull" {
		return a
	}
	a = a.setFilter(pullFilter{row.it.pull})
	a.growFrom = row.it.pull
	return a
}

func (a App) grown(row sideRow) App {
	if a.growFrom != 0 && row.it.kind == "pull" {
		a.filter = pullRun(stackPulls(*a.review), a.growFrom, row.it.pull)
	}
	return a
}

func (a App) growingInReview() App {
	if a.focus != reviewPanel || a.screen == screenHome {
		a.growFrom = 0
	}
	return a
}

func (a App) keptFilter() App {
	stack := stackPulls(*a.review)
	a.filter = a.filter.within(stack)
	if !containsInt(stack, a.growFrom) {
		a.growFrom = 0
	}
	return a
}
