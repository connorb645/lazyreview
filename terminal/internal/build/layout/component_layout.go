package layout

import (
	"slices"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var componentLayers = arrangement{layerGap: 90, nodeGap: 70, partGap: 70, linkedAspect: 2.4, packedGap: 70, packedAspect: 1.6}

func layoutComponent(tr *i18n.TranslationSet, c models.Component, m models.Model, nodes map[string]models.Node, groups map[string]models.Group) laid {
	var order []string
	pkgs := map[string]laid{}
	for _, p := range c.Packages {
		if l, ok := layoutPackage(c, p, m, nodes, groups); ok {
			order = append(order, p.ID)
			pkgs[p.ID] = l
		}
	}
	tests := testPackages(c, m)
	homeOf := func(id string) (home, bool) {
		n, ok := nodes[id]
		if g, grouped := groups[id]; grouped && len(g.Members) > 0 {
			n, ok = nodes[g.Members[0]]
		}
		return home{component: n.Component, pkg: n.Package}, ok
	}
	var pairs []*packagePair
	byKey := map[string]*packagePair{}
	for _, e := range m.Edges {
		a, aok := homeOf(e.Source)
		b, bok := homeOf(e.Target)
		if !aok || !bok || a.component != c.ID || b.component != c.ID || a.pkg == b.pkg || tests[a.pkg] || tests[b.pkg] {
			continue
		}
		key := a.pkg + ">" + b.pkg
		p, ok := byKey[key]
		if !ok {
			p = &packagePair{from: a.pkg, to: b.pkg}
			byKey[key] = p
			pairs = append(pairs, p)
		}
		p.count++
		p.links = append(p.links, [2]string{e.Source, e.Target})
		if e.Kind == "test" {
			p.tests++
		}
		if e.Key && e.Label != "" {
			p.labels = append(p.labels, e.Label)
		}
	}
	var arrows []models.PackageArrow
	var links [][2]string
	for _, p := range pairs {
		if _, ok := pkgs[p.from]; !ok {
			continue
		}
		if _, ok := pkgs[p.to]; !ok {
			continue
		}
		var labels []string
		for _, l := range p.labels {
			if !slices.Contains(labels, l) {
				labels = append(labels, l)
			}
		}
		words := strings.Join(labels[:min(2, len(labels))], ", ")
		if words == "" && p.tests == p.count {
			words = tr.DiagramTests
		}
		arrows = append(arrows, models.PackageArrow{From: p.from, To: p.to, Words: words, CountText: utils.Plural(p.count, tr.LinkNoun), Links: p.links})
		links = append(links, [2]string{p.from, p.to})
	}
	var items []item
	for _, id := range order {
		items = append(items, item{id: id, w: pkgs[id].w, h: pkgs[id].h})
	}
	content := arrange(items, links, componentLayers)
	top := float64(componentHead + 16)
	w, h := content.w+72, content.h+top+36
	needed := max(460,
		measure(c.Title, "componentTitle")+measure(tr.DiagramOpenDiff, "componentOpen")+80,
		measure(c.Tag+" · "+componentCounts(tr, m, c), "componentTag")+56)
	if len(m.Components) > 1 {
		needed = max(needed, farTitleRoom(c.Title))
	}
	dx := 0.0
	if w < needed {
		dx, w = (needed-w)/2, needed
	}
	boxes := map[string]box{}
	for _, id := range order {
		at := content.at[id]
		frame := box{x: 36 + at.x + dx, y: top + at.y, w: at.w, h: at.h}
		boxes["pkg:"+c.ID+":"+id] = frame
		for bid, b := range pkgs[id].boxes {
			boxes[bid] = box{x: b.x + frame.x, y: b.y + frame.y, w: b.w, h: b.h}
		}
	}
	return laid{w: w, h: h, boxes: boxes, arrows: arrows}
}

func testPackages(c models.Component, m models.Model) map[string]bool {
	tests := map[string]bool{}
	for _, p := range c.Packages {
		holds, all := false, true
		for _, n := range m.Nodes {
			if n.Component == c.ID && n.Package == p.ID {
				holds, all = true, all && n.Test
			}
		}
		tests[p.ID] = holds && all
	}
	return tests
}

func componentCounts(tr *i18n.TranslationSet, m models.Model, c models.Component) string {
	if c.Context {
		count := 0
		for _, f := range m.ContextFiles {
			if f.Component == c.ID {
				count++
			}
		}
		return utils.ResolvePlaceholderString(tr.DiagramUnchangedFiles, map[string]string{"count": utils.Itoa(count)})
	}
	count, adds, dels := 0, 0, 0
	for _, f := range m.Files {
		if f.Component == c.ID {
			count, adds, dels = count+1, adds+f.Adds, dels+f.Dels
		}
	}
	text := utils.ResolvePlaceholderString(tr.DiagramChangedFiles, map[string]string{"count": utils.Itoa(count), "adds": grouped(adds)})
	if dels != 0 {
		text += utils.ResolvePlaceholderString(tr.DiagramRemovedLines, map[string]string{"dels": grouped(dels)})
	}
	return text
}
