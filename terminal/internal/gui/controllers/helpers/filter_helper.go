package helpers

import (
	"slices"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// FilterHelper sets the pull request filter of a stack: one pull request,
// all of them, or a highlight grown over a run of them.
type FilterHelper struct {
	c *HelperCommon
}

// FilterBy is the state with the filter on the pull request by pull
// requests on from it, through All, going round at the ends.
func (self *FilterHelper) FilterBy(s context.ContextTree, by int) context.ContextTree {
	if s.Review == nil || s.Review.Page.Stack == nil {
		return s.Said(self.c.Tr.OnlyStackFilters, true)
	}
	order := append([]int{0}, presentation.StackPulls(*s.Review)...)
	from := 0
	if len(s.Filter) > 0 {
		from = s.Filter[0]
		if by > 0 {
			from = s.Filter[len(s.Filter)-1]
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
		return self.SetFilter(s, nil)
	}
	return self.SetFilter(s, types.PullFilter{next})
}

// SetFilter is the state with a filter set and no highlight growing.
func (self *FilterHelper) SetFilter(s context.ContextTree, f types.PullFilter) context.ContextTree {
	s.Filter, s.GrowFrom = f, 0
	return self.c.Helpers.Refresh.Relaid(s)
}

// PullCommand is the state with the filter that :pull names: all, or a
// pull request of the stack.
func (self *FilterHelper) PullCommand(s context.ContextTree, arg string) context.ContextTree {
	if s.Review == nil || s.Review.Page.Stack == nil {
		return s.Said(self.c.Tr.OnlyStackFilters, true)
	}
	if arg == "all" {
		return self.SetFilter(s, nil)
	}
	n, err := strconv.Atoi(strings.TrimPrefix(arg, "#"))
	if err != nil || presentation.StackMember(s.Review.Page.Stack, n).Head == "" {
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.NoPullInStack, map[string]string{"pull": arg}), true)
	}
	return self.SetFilter(s, types.PullFilter{n})
}

// Highlight is the state with a highlight started on a pull request row of
// a stack, to grow from.
func (self *FilterHelper) Highlight(s context.ContextTree, row presentation.SideRow) context.ContextTree {
	if s.Review.Page.Stack == nil || row.Item.Kind != "pull" {
		return s
	}
	s = self.SetFilter(s, types.PullFilter{row.Item.Pull})
	s.GrowFrom = row.Item.Pull
	return s
}

// Grown is the state with a growing highlight reaching to a pull request
// row.
func (self *FilterHelper) Grown(s context.ContextTree, row presentation.SideRow) context.ContextTree {
	if s.GrowFrom != 0 && row.Item.Kind == "pull" {
		s.Filter = pullRun(presentation.StackPulls(*s.Review), s.GrowFrom, row.Item.Pull)
	}
	return s
}

// GrowingInReview is the state with the highlight no longer growing once
// the keys leave the Review panel.
func (self *FilterHelper) GrowingInReview(s context.ContextTree) context.ContextTree {
	if s.Focus() != context.ReviewPanel || s.Screen() == context.ScreenHome {
		s.GrowFrom = 0
	}
	return s
}

// KeptFilter is the state with the filter kept to the pull requests the
// stack still holds.
func (self *FilterHelper) KeptFilter(s context.ContextTree) context.ContextTree {
	stack := presentation.StackPulls(*s.Review)
	s.Filter = s.Filter.Within(stack)
	if !slices.Contains(stack, s.GrowFrom) {
		s.GrowFrom = 0
	}
	return s
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
