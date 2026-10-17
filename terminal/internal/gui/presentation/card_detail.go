package presentation

import (
	"github.com/connorb645/lazyreview/terminal/internal/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

// CardDetail is a card's second line under a filter: a component's or a
// package's count of what the filter shows, or the card's own detail.
func CardDetail(tr *i18n.TranslationSet, d types.ReviewData, l diagram.Level, card diagram.Card, filter types.PullFilter) string {
	for _, c := range d.Page.Model.Components {
		switch {
		case card.Kind == "component" && c.ID == card.ID:
			return diagram.ComponentDetail(tr, d.Page, c, filter.ShowsAny, filter.Lines)
		case card.Kind == "package" && c.ID == l.Component:
			for _, p := range c.Packages {
				if p.ID == card.ID {
					return diagram.PackageDetail(tr, d.Page, c, p, filter.ShowsAny)
				}
			}
		}
	}
	return card.Detail
}
