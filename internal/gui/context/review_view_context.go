package context

import (
	"time"

	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/theme"
	"github.com/connorb645/lazyreview/internal/utils"
)

// ReviewViewContext is the Review view in the main panel: its lines, the
// sections folded, the conversations opened, by their keys, and the one the
// Notes panel opened only to show it.
type ReviewViewContext struct {
	View   Scroller
	Folded map[string]bool
	Opened map[string]bool
	Peeked string
}

// Rebuilt is the view drawn again from the review in a width, its words
// naming keys as labels label them, its selection kept, shown in a height.
func (v ReviewViewContext) Rebuilt(tr *i18n.TranslationSet, th theme.Theme, md *presentation.Markdown, labels presentation.KeyLabels, loc *time.Location, d types.ReviewData, filter types.PullFilter, width, height int) ReviewViewContext {
	lines, items := presentation.BuildReview(tr, th, md, labels, loc, d, filter, v.Folded, v.Opened, width)
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

// Opening is the view with a conversation, by its key, opened or folded.
func (v ReviewViewContext) Opening(key string, open bool) ReviewViewContext {
	v.Opened = utils.CopyBools(v.Opened)
	v.Opened[key] = open
	return v
}

// Peeking is the view with a conversation, by its key, open to show it, and
// the one it last opened that way folded again. One already open stays as
// it was, and is not folded when another is shown.
func (v ReviewViewContext) Peeking(key string) ReviewViewContext {
	if v.Peeked == key {
		return v
	}
	v = v.Unpeeked()
	if !v.Opened[key] {
		v = v.Opening(key, true)
		v.Peeked = key
	}
	return v
}

// Unpeeked is the view with the conversation it opened only to show folded
// again.
func (v ReviewViewContext) Unpeeked() ReviewViewContext {
	if v.Peeked != "" {
		v = v.Opening(v.Peeked, false)
		v.Peeked = ""
	}
	return v
}

// Kept is the view with a conversation, by its key, left as it is folded
// now, since it was folded or opened on purpose, when another is shown.
func (v ReviewViewContext) Kept(key string) ReviewViewContext {
	if v.Peeked == key {
		v.Peeked = ""
	}
	return v
}

// OnConversation is the view with its selection on a conversation's item,
// by its number, shown in a height, or as it was with no such item.
func (v ReviewViewContext) OnConversation(id, height int) ReviewViewContext {
	for i, it := range v.View.Items {
		if it.Kind == "conversation" && it.Thread == id {
			v.View = v.View.To(i, height)
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
