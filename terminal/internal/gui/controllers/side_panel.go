package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

func sidePanelBindings(c *helpers.HelperCommon) []Binding {
	tr, h, kb := c.Tr, c.Helpers, c.Config.Keybinding
	down := sideMoved(h, func(side context.SideContext, count Count, height int) context.SideContext {
		return side.Move(count.N, height)
	})
	up := sideMoved(h, func(side context.SideContext, count Count, height int) context.SideContext {
		return side.Move(-count.N, height)
	})
	top := sideMoved(h, func(side context.SideContext, count Count, height int) context.SideContext {
		if count.Typed {
			return side.To(count.N-1, height)
		}
		return side.To(0, height)
	})
	bottom := sideMoved(h, func(side context.SideContext, count Count, height int) context.SideContext {
		if count.Typed {
			return side.To(count.N-1, height)
		}
		return side.To(side.Landing(len(side.Rows)-1, -1), height)
	})
	paged := func(half, up bool) Handler {
		return sideMoved(h, func(side context.SideContext, count Count, height int) context.SideContext {
			by := pageBy(height, half, up) * count.N
			return side.Scroll(by, height).Move(by, height)
		})
	}
	return withReviewAll([]Binding{
		{Action: "down", Keys: kb.Panels.Down, Handler: down, Description: tr.Down},
		{Action: "up", Keys: kb.Panels.Up, Handler: up, Description: tr.Up},
		{Action: "top", Keys: kb.Panels.Top, Handler: top, Description: tr.Top},
		{Action: "bottom", Keys: kb.Panels.Bottom, Handler: bottom, Description: tr.Bottom},
		{Action: "half down", Keys: kb.Panels.HalfDown, Handler: paged(true, false), Description: tr.HalfPageDown},
		{Action: "half up", Keys: kb.Panels.HalfUp, Handler: paged(true, true), Description: tr.HalfPageUp},
		{Action: "page down", Keys: kb.Panels.PageDown, Handler: paged(false, false), Description: tr.PageDown},
		{Action: "page up", Keys: kb.Panels.PageUp, Handler: paged(false, true), Description: tr.PageUp},
		{Action: "open", Keys: kb.Panels.Open, Handler: uncounted(func(s context.ContextTree) (context.ContextTree, tea.Cmd) {
			return openSide(c, s)
		}), Description: tr.Open},
		{Action: "previous panel", Keys: kb.Panels.PreviousPanel, Handler: uncounted(func(s context.ContextTree) (context.ContextTree, tea.Cmd) {
			return h.Panels.FocusBy(s, -1)
		}), Description: tr.PreviousPanel},
		{Action: "next panel", Keys: kb.Panels.NextPanel, Handler: uncounted(func(s context.ContextTree) (context.ContextTree, tea.Cmd) {
			return h.Panels.FocusBy(s, 1)
		}), Description: tr.NextPanel},
	})
}

func sideMoved(h *helpers.Helpers, move func(side context.SideContext, count Count, height int) context.SideContext) Handler {
	return func(s context.ContextTree, count Count) (context.ContextTree, tea.Cmd) {
		n := s.Focus()
		s = s.WithSide(n, move(s.Side(n), count, h.Window.SideHeight(s, n)))
		return h.Panels.Followed(h.Files.FromCursor(h.Filter.FromCursor(s)))
	}
}

func openSide(c *helpers.HelperCommon, s context.ContextTree) (context.ContextTree, tea.Cmd) {
	h := c.Helpers
	row, ok := s.Side(s.Focus()).Selected()
	if !ok {
		return s, nil
	}
	if s.Focus() == context.FilesPanel && folds(row) {
		return h.Panels.TreeFolded(s, row)
	}
	s = h.Jump.Leave(s)
	s = s.Focused(context.MainPanel, s.Driving())
	return h.Refresh.Relaid(s), nil
}

func folds(row presentation.SideRow) bool {
	return row.Item.Kind == "component" || row.Item.Kind == "dir"
}

func selectedRow(f func(s context.ContextTree, row presentation.SideRow) (context.ContextTree, tea.Cmd)) Handler {
	return func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
		row, ok := s.Side(s.Focus()).Selected()
		if !ok {
			return s, nil
		}
		return f(s, row)
	}
}

func panelsHint(c *helpers.HelperCommon, labels presentation.KeyLabels) [2]string {
	return [2]string{labels.Of("previous panel", "next panel"), c.Tr.Panels}
}

func sideMoveKeys(labels presentation.KeyLabels) string {
	return labels.Of("down", "up", "top", "bottom")
}

func pageBy(height int, half, up bool) int {
	by := height
	if half {
		by = max(1, height/2)
	}
	if up {
		return -by
	}
	return by
}
