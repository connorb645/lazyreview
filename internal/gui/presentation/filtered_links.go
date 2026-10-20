package presentation

import (
	"strings"

	"github.com/connorb645/lazyreview/internal/diagram"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

// FilteredLinks are a level's links under a filter: each link between
// components or packages worded with the count of its links the filter
// shows, and out when that is none of them; a link between types out when
// the filter leaves out either end.
func FilteredLinks(tr *i18n.TranslationSet, d types.ReviewData, links []diagram.Link, filter types.PullFilter) []diagram.Link {
	if len(filter) == 0 {
		return links
	}
	out := make([]diagram.Link, len(links))
	for i, l := range links {
		out[i] = l
		if l.Kind != "summary" && l.Kind != "package" {
			out[i].Out = l.Outside == "" && !(Shows(d, l.From, filter) && Shows(d, l.To, filter))
			continue
		}
		if len(l.Ways) == 0 {
			continue
		}
		var words []string
		shown := 0
		for _, w := range l.Ways {
			n := 0
			for _, link := range w.Links {
				if Shows(d, link[0], filter) && Shows(d, link[1], filter) {
					n++
				}
			}
			shown += n
			words = append(words, wayWords(tr, w, n))
		}
		if l.Words != "" {
			out[i].Words = strings.Join(words, " · ")
		}
		out[i].Out = shown == 0
	}
	return out
}

// Shows says whether a filter shows a type or a group of types: unchanged
// code always, a type a pull request it holds changed, and a group one of
// its members.
func Shows(d types.ReviewData, id string, filter types.PullFilter) bool {
	m := d.Page.Model
	if group, ok := strings.CutPrefix(id, "group:"); ok {
		for _, g := range m.Groups {
			if g.ID == group {
				for _, member := range g.Members {
					if Shows(d, member, filter) {
						return true
					}
				}
				return false
			}
		}
		return true
	}
	for _, n := range m.Nodes {
		if n.ID == id {
			return n.State == "context" || filter.ShowsAny(n.Pulls)
		}
	}
	return true
}

func wayWords(tr *i18n.TranslationSet, w models.ArrowWay, n int) string {
	words := utils.Plural(n, tr.LinkNoun)
	if w.Label != "" {
		words = w.Label + " · " + words
	}
	if w.Arrow != "" {
		words = w.Arrow + " " + words
	}
	return words
}
