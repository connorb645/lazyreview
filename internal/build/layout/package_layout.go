package layout

import (
	"math"
	"slices"

	"github.com/connorb645/lazyreview/internal/models"
)

var (
	packageLayers = arrangement{layerGap: 40, nodeGap: 20, partGap: 26, linkedAspect: 1.7, packedGap: 20, packedAspect: 1.9}
	groupRows     = arrangement{packedGap: 12, packedAspect: 1.9}
)

func atRest(e models.Edge) bool {
	switch e.Kind {
	case "realize", "inherit", "has", "extend":
		return true
	}
	return e.Kind == "use" && e.Key
}

func upward(e models.Edge) bool {
	return e.Kind == "realize" || e.Kind == "inherit" || e.Kind == "extend"
}

func layoutPackage(c models.Component, p models.Package, m models.Model, nodes map[string]models.Node, groups map[string]models.Group) (laid, bool) {
	var items []item
	var groupIDs []string
	ids := map[string]bool{}
	inside := map[string]placed{}
	tops := map[string]float64{}
	for _, n := range m.Nodes {
		if n.Component != c.ID || n.Package != p.ID {
			continue
		}
		if n.Group == "" {
			w, h := nodeSize(n)
			items = append(items, item{id: n.ID, w: w, h: h})
			ids[n.ID] = true
		} else if id := "group:" + n.Group; !slices.Contains(groupIDs, id) {
			groupIDs = append(groupIDs, id)
			ids[id] = true
		}
	}
	if len(ids) == 0 {
		return laid{}, false
	}
	for _, id := range groupIDs {
		g := groups[id]
		var members []item
		for _, member := range g.Members {
			if n, ok := nodes[member]; ok {
				w, h := nodeSize(n)
				members = append(members, item{id: member, w: w, h: h})
			}
		}
		tops[id] = 34
		caption := measure(g.Title, "groupTitle")
		if g.Description != "" {
			tops[id] = 48
			caption = max(caption, measure(g.Description, "groupNote"))
		}
		inside[id] = arrange(members, nil, groupRows)
		items = append(items, item{id: id, w: max(inside[id].w+28, math.Ceil(caption+28)), h: max(inside[id].h+tops[id]+14, 60)})
	}
	laidAs := func(id string) string {
		if n, ok := nodes[id]; ok && n.Group != "" {
			return "group:" + n.Group
		}
		return id
	}
	var links [][2]string
	taken := map[string]bool{}
	for _, e := range m.Edges {
		if !atRest(e) {
			continue
		}
		s, t := laidAs(e.Source), laidAs(e.Target)
		if s == t || !ids[s] || !ids[t] || taken[s+">"+t] {
			continue
		}
		taken[s+">"+t] = true
		if upward(e) {
			links = append(links, [2]string{t, s})
		} else {
			links = append(links, [2]string{s, t})
		}
	}
	head := 0.0
	switch {
	case p.Bare:
	case p.Description != "":
		head = packageHeadWithNote
	default:
		head = packageHead
	}
	content := arrange(items, links, packageLayers)
	w, h := content.w+36, content.h+head+10+18
	needed := 0.0
	if !p.Bare {
		needed = measure(p.Title, "packageTitle") + measure(p.Path, "packagePath") + 64
	}
	if p.Description != "" {
		needed = max(needed, measure(p.Description, "packageNote")+40)
	}
	dx := 0.0
	if w < needed {
		dx, w = (needed-w)/2, needed
	}
	boxes := map[string]box{}
	for _, it := range items {
		at := content.at[it.id]
		b := box{x: 18 + at.x + dx, y: head + 10 + at.y, w: at.w, h: at.h}
		boxes[it.id] = b
		for member, at := range inside[it.id].at {
			boxes[member] = box{x: b.x + 14 + at.x, y: b.y + tops[it.id] + at.y, w: at.w, h: at.h}
		}
	}
	return laid{w: w, h: h, boxes: boxes}, true
}
