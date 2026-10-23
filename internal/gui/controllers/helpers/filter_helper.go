package helpers

import (
	"slices"
	"strings"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/utils"
)

// FilterHelper sets the pull request filter of a stack: one pull request,
// all of them, or a highlight grown over a run of them. The Review panel's
// cursor and the filter follow each other.
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
// member of the stack, a pull request by its number or a branch by its
// name.
func (self *FilterHelper) PullCommand(s context.ContextTree, arg string) context.ContextTree {
	if s.Review == nil || s.Review.Page.Stack == nil {
		return s.Said(self.c.Tr.OnlyStackFilters, true)
	}
	if arg == "all" {
		return self.SetFilter(s, nil)
	}
	pull, found := presentation.MemberNamed(*s.Review, arg)
	switch {
	case !found && s.Review.State.Kind == "branch":
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.NoBranchInStack, map[string]string{"branch": arg}), true)
	case !found:
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.NoPullInStack, map[string]string{"pull": strings.TrimPrefix(arg, "#")}), true)
	}
	return self.SetFilter(s, types.PullFilter{pull})
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

// FromCursor is the state with the filter on what the Review panel's
// cursor is on, while that panel has the keys: a growing highlight reaching
// to its pull request, the pull request alone, or the whole stack on All. A
// growing highlight keeps the cursor on the pull requests.
func (self *FilterHelper) FromCursor(s context.ContextTree) context.ContextTree {
	if s.Focus() != context.ReviewPanel || s.Review == nil || s.Review.Page.Stack == nil {
		return s
	}
	side := s.Side(context.ReviewPanel)
	row, ok := side.Selected()
	if ok && s.GrowFrom != 0 && row.Item.Kind != "pull" {
		side = side.To(side.Cursor+1, self.c.Helpers.Window.SideHeight(s, context.ReviewPanel))
		s = s.WithSide(context.ReviewPanel, side)
		row, ok = side.Selected()
	}
	switch {
	case !ok:
	case s.GrowFrom != 0:
		s.Filter = pullRun(presentation.StackPulls(*s.Review), s.GrowFrom, row.Item.Pull)
	case row.Item.Kind == "pull":
		s.Filter = types.PullFilter{row.Item.Pull}
	default:
		s.Filter = nil
	}
	return s
}

// OnFilter is the state with a stack's Review panel's cursor on the
// filter: on All while it shows the whole stack, else kept on a pull
// request it holds, or on its last.
func (self *FilterHelper) OnFilter(s context.ContextTree) context.ContextTree {
	if s.Review == nil || s.Review.Page.Stack == nil {
		return s
	}
	side := s.Side(context.ReviewPanel)
	key := presentation.AllKey
	if len(s.Filter) > 0 {
		if row, ok := side.Selected(); ok && row.Item.Kind == "pull" && slices.Contains(s.Filter, row.Item.Pull) {
			return s
		}
		key = presentation.PullKey(s.Filter[len(s.Filter)-1])
	}
	return s.WithSide(context.ReviewPanel, side.To(side.Index(key), self.c.Helpers.Window.SideHeight(s, context.ReviewPanel)))
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
