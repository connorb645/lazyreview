package gui

import (
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
)

type reviewView struct {
	view   scroller
	folded map[string]bool
}

func (v reviewView) rebuilt(tr *i18n.TranslationSet, th theme.Theme, md *presentation.Markdown, loc *time.Location, d types.ReviewData, filter types.PullFilter, width, height int) reviewView {
	lines, items := presentation.BuildReview(tr, th, md, loc, d, filter, v.folded, width)
	v.view = v.view.reset(lines, items, sameItem, height)
	return v
}

func (v reviewView) section(back bool, count, height int) reviewView {
	for ; count > 0; count-- {
		for i := v.view.cursor + step(back); i >= 0 && i < len(v.view.items); i += step(back) {
			if v.view.items[i].Kind == "section" {
				v.view = v.view.to(i, height)
				break
			}
		}
	}
	return v
}

func (v reviewView) sectionAt() string {
	if v.view.cursor < len(v.view.items) {
		return v.view.items[v.view.cursor].Section
	}
	return ""
}

func step(back bool) int {
	if back {
		return -1
	}
	return 1
}

func sameItem(a, b presentation.Item) bool {
	return a.Kind == b.Kind && a.Section == b.Section && a.Component == b.Component && a.Pull == b.Pull && a.Comment == b.Comment && a.URL == b.URL && samePlace(a, b)
}

func samePlace(a, b presentation.Item) bool {
	if a.Place == nil || b.Place == nil {
		return a.Place == b.Place
	}
	return *a.Place == *b.Place
}
