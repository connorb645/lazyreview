package gui

import "github.com/connorb645/lazyreview/terminal/internal/gui/context"

func (a App) menuOpen() bool {
	return a.Has(context.MenuKey) || a.Has(context.FinderKey)
}

func (a App) withMenu(m context.MenuContext) App {
	key := context.MenuKey
	if m.Finding {
		key = context.FinderKey
	}
	a.ContextTree = a.Opened(key)
	a.Menu = m
	return a
}

func (a App) withoutMenu() App {
	a.ContextTree = a.Closed(context.MenuKey).Closed(context.FinderKey)
	return a
}

func (a App) popped() bool {
	return a.menuOpen() || a.Has(context.PostingKey) || a.Has(context.ConfirmationKey)
}
