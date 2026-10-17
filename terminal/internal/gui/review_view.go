package gui

import (
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

type reviewView struct {
	view   scroller
	folded map[string]bool
}

func (v reviewView) rebuilt(tr *i18n.TranslationSet, d reviewData, filter pullFilter, width, height int, p palette, md *markdown, loc *time.Location) reviewView {
	lines, items := buildReview(tr, d, filter, v.folded, width, p, md, loc)
	v.view = v.view.reset(lines, items, sameItem, height)
	return v
}

func (v reviewView) section(back bool, count, height int) reviewView {
	for ; count > 0; count-- {
		for i := v.view.cursor + step(back); i >= 0 && i < len(v.view.items); i += step(back) {
			if v.view.items[i].kind == "section" {
				v.view = v.view.to(i, height)
				break
			}
		}
	}
	return v
}

func (v reviewView) sectionAt() string {
	if v.view.cursor < len(v.view.items) {
		return v.view.items[v.view.cursor].section
	}
	return ""
}

func step(back bool) int {
	if back {
		return -1
	}
	return 1
}

func sameItem(a, b item) bool {
	return a.kind == b.kind && a.section == b.section && a.component == b.component && a.pull == b.pull && a.comment == b.comment && a.url == b.url && samePlace(a, b)
}

func samePlace(a, b item) bool {
	if a.place == nil || b.place == nil {
		return a.place == b.place
	}
	return *a.place == *b.place
}
