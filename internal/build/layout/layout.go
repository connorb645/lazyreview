// Package layout places the diagram's parts for a review's page: the types
// in each package, the packages in each component, and the components, in
// their areas, on the canvas, with the arrows between packages and between
// components. Linked types sit in layers, supertypes above their subtypes
// and holders above what they hold; types nothing links sit in tidy rows.
package layout

import (
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
)

// Of is where the diagram places each part of model: each component, each
// package frame, type and group, and the arrows between packages and
// between components with their words. The same model always gives the
// same places.
func Of(tr *i18n.TranslationSet, model models.Model) models.Layout {
	d := place(tr, model)
	out := models.Layout{Components: map[string]models.Box{}, Boxes: map[string]models.Box{}, PkgArrows: d.arrows, Summaries: d.summaries}
	for id, b := range d.components {
		out.Components[id] = b.rounded()
	}
	for id, b := range d.boxes {
		out.Boxes[id] = b.rounded()
	}
	return out
}

func place(tr *i18n.TranslationSet, model models.Model) drawn {
	nodes := map[string]models.Node{}
	for _, n := range model.Nodes {
		nodes[n.ID] = n
	}
	groups := map[string]models.Group{}
	for _, g := range model.Groups {
		groups["group:"+g.ID] = g
	}
	results := map[string]laid{}
	for _, c := range model.Components {
		results[c.ID] = layoutComponent(tr, c, model, nodes, groups)
	}
	return layoutCanvas(tr, model, results)
}
