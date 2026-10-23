package context

import (
	"time"

	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/theme"
)

// ReviewViewContext is the Review view in the main panel: its lines and
// the sections folded.
type ReviewViewContext struct {
	View   Scroller
	Folded map[string]bool
}

// Rebuilt is the view drawn again from the review in a width, its words
// naming keys as labels label them, its selection kept, shown in a height.
func (v ReviewViewContext) Rebuilt(tr *i18n.TranslationSet, th theme.Theme, md *presentation.Markdown, labels presentation.KeyLabels, loc *time.Location, d types.ReviewData, filter types.PullFilter, width, height int) ReviewViewContext {
	lines, items := presentation.BuildReview(tr, th, md, labels, loc, d, filter, v.Folded, width)
	v.View = v.View.Reset(lines, items, sameItem, height)
	return v
}

// Section is the view with the next or previous section selected, count
// times over.
func (v ReviewViewContext) Section(back bool, count, height int) ReviewViewContext {
	for ; count > 0; count-- {
		for i := v.View.Cursor + step(back); i >= 0 && i < len(v.View.Items); i += step(back) {
			if v.View.Items[i].Kind == "section" {
				v.View = v.View.To(i, height)
				break
			}
		}
	}
	return v
}

// SectionAt is the id of the section the selection is in.
func (v ReviewViewContext) SectionAt() string {
	if v.View.Cursor < len(v.View.Items) {
		return v.View.Items[v.View.Cursor].Section
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
	return a.Kind == b.Kind && a.Section == b.Section && a.Component == b.Component && a.Pull == b.Pull && a.Comment == b.Comment && a.Thread == b.Thread && a.URL == b.URL && samePlace(a, b)
}

func samePlace(a, b presentation.Item) bool {
	if a.Place == nil || b.Place == nil {
		return a.Place == b.Place
	}
	return *a.Place == *b.Place
}
