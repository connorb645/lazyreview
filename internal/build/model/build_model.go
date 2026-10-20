package model

import (
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/internal/build/components"
	"github.com/connorb645/lazyreview/internal/build/diff"
	"github.com/connorb645/lazyreview/internal/build/placement"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

const groupAt = 6

var notIDText = regexp.MustCompile(`[^a-z0-9]+`)

// BuildModel puts the diagram's model together from the diff's files, every
// node to draw (changed, unchanged in a changed file, or unchanged context),
// the files' imports and the config: the components and their packages, the
// nodes placed in them, the links between them, groups of look-alike types
// and the links between components. The heading is left for the page.
func BuildModel(tr *i18n.TranslationSet, files []diff.File, given []*Node, imports []FileImports, config components.Config, rootTitle string) models.Model {
	nodes := make([]*Node, len(given))
	for i, n := range given {
		copied := *n
		nodes[i] = &copied
	}
	assignIDs(nodes)

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	changed := placement.PlaceFiles(tr, paths, config, false, rootTitle)
	var contextPaths []string
	for _, n := range nodes {
		if n.State == "context" && !slices.Contains(contextPaths, n.File) {
			contextPaths = append(contextPaths, n.File)
		}
	}
	context := placement.PlaceFiles(tr, contextPaths, config, true, tr.DiagramRepositoryRoot)
	for _, n := range nodes {
		place, held := changed.Places[n.File]
		if !held {
			place = context.Places[n.File]
		}
		n.Component, n.Package = place.Component, place.Package
	}

	edges := FindRelations(tr, nodes, imports)
	byID := map[string]*Node{}
	for _, n := range nodes {
		byID[n.ID] = n
	}
	groups := makeGroups(tr, nodes, edges, config, byID)
	edges = regroupEdges(tr, edges, groups, byID)
	edges = applyLinks(edges, config.Links, nodes, groups)

	placed := colorComponents(append(append([]placement.Component{}, changed.Components...), context.Components...))
	ids := map[string]bool{}
	for _, c := range placed {
		ids[c.ID] = true
	}
	var areas []models.Area
	areaOf := map[string]models.Area{}
	for _, a := range config.Areas {
		area := models.Area{ID: a.ID, Title: a.Title, Description: a.Description}
		for _, id := range a.Components {
			if ids[id] {
				area.Components = append(area.Components, id)
			}
		}
		if len(area.Components) == 0 {
			continue
		}
		areas = append(areas, area)
		for _, id := range area.Components {
			areaOf[id] = area
		}
	}

	names := map[string]bool{}
	for _, n := range nodes {
		names[n.Name] = true
	}
	for _, n := range nodes {
		n.Stereotype = stereotypeOf(tr, n, names)
	}
	componentOf := func(id string) string {
		if name, isGroup := strings.CutPrefix(id, "group:"); isGroup {
			for _, g := range groups {
				if g.id == name {
					return g.component
				}
			}
		}
		return byID[id].Component
	}

	model := models.Model{Areas: orNone(areas), Edges: orNone(edges), Summary: summarize(edges, componentOf, config.Summaries, len(placed))}
	for _, c := range placed {
		component := models.Component{ID: c.ID, Title: c.Title, Description: c.Description, Tag: c.Tag, Color: c.Color, Context: c.Context}
		if area, held := areaOf[c.ID]; held {
			component.Area = area.ID
		}
		for _, p := range c.Packages {
			bare := len(c.Packages) == 1 && p.Description == "" && (p.Title == c.Title || p.Path == c.Tag)
			component.Packages = append(component.Packages, models.Package{ID: p.ID, Title: p.Title, Description: p.Description, Path: p.Path, Bare: bare})
		}
		model.Components = append(model.Components, component)
	}
	for _, f := range files {
		place := changed.Places[f.Path]
		model.Files = append(model.Files, models.FileEntry{
			Path: f.Path, OldPath: f.OldPath, Status: f.Status, Adds: f.Adds, Dels: f.Dels, Binary: f.Binary, Collapsed: f.Collapsed,
			Pulls: f.Pulls, ByPull: f.ByPull, Component: place.Component, Package: place.Package,
		})
	}
	model.ContextFiles = []models.FileEntry{}
	for _, path := range contextPaths {
		place := context.Places[path]
		model.ContextFiles = append(model.ContextFiles, models.FileEntry{Path: path, Component: place.Component, Package: place.Package})
	}
	for _, n := range nodes {
		line := n.Line
		if line == unplaced {
			line = 1
		}
		model.Nodes = append(model.Nodes, models.Node{
			ID: n.ID, Name: n.Name, Kind: n.Kind, Stereotype: n.Stereotype, File: n.File, Line: line,
			Component: n.Component, Package: n.Package, State: n.State, Group: n.Group, Test: n.Test, Pulls: n.Pulls, Properties: n.Properties,
		})
	}
	model.Groups = []models.Group{}
	for _, g := range groups {
		model.Groups = append(model.Groups, models.Group{ID: g.id, Title: g.title, Description: g.description, Members: g.members})
	}
	return model
}

func assignIDs(nodes []*Node) {
	count := map[string]int{}
	for _, n := range nodes {
		if n.Kind != "artifact" && n.Kind != "extension" {
			count[n.Name]++
		}
	}
	taken := map[string]bool{}
	for _, n := range nodes {
		var id string
		switch {
		case n.Kind == "artifact":
			id = "file:" + n.File
		case n.Kind == "extension":
			id = "ext:" + n.File
		case count[n.Name] > 1:
			id = n.Name + "@" + n.File
		default:
			id = n.Name
		}
		for k := 2; taken[id]; k++ {
			id = id + "#" + strconv.Itoa(k)
		}
		taken[id] = true
		n.ID = id
	}
}

func makeGroups(tr *i18n.TranslationSet, nodes []*Node, edges []models.Edge, config components.Config, byID map[string]*Node) []group {
	var groups []group
	grouped := map[string]bool{}
	var links []models.Edge
	for _, e := range edges {
		if e.Kind == "realize" || e.Kind == "inherit" {
			links = append(links, e)
		}
	}
	add := func(id, title, description string, members []string, target, kind string) {
		if len(members) < 2 {
			return
		}
		first := byID[members[0]]
		for _, m := range members {
			byID[m].Group = id
			grouped[m] = true
		}
		title = utils.ResolvePlaceholderString(tr.DiagramGroup, map[string]string{"count": strconv.Itoa(len(members)), "title": title})
		groups = append(groups, group{id: id, title: title, description: description, members: members, target: target, kind: kind, component: first.Component, pkg: first.Package})
	}

	for _, g := range config.Groups {
		var target *Node
		for _, n := range nodes {
			if n.Name == g.Of && n.Kind != "artifact" {
				target = n
				break
			}
		}
		var keys []string
		byPackage := map[string][]models.Edge{}
		for _, e := range links {
			if target == nil || e.Target != target.ID || grouped[e.Source] {
				continue
			}
			n := byID[e.Source]
			key := n.Component + "/" + n.Package
			if _, seen := byPackage[key]; !seen {
				keys = append(keys, key)
			}
			byPackage[key] = append(byPackage[key], e)
		}
		var best []models.Edge
		for _, key := range keys {
			if len(byPackage[key]) > len(best) {
				best = byPackage[key]
			}
		}
		members := make([]string, len(best))
		for i, e := range best {
			members[i] = e.Source
		}
		targetID, kind := "", "realize"
		if target != nil {
			targetID = target.ID
		}
		if len(best) > 0 {
			kind = best[0].Kind
		}
		add(g.ID, g.Title, g.Description, members, targetID, kind)
	}

	type bucket struct{ place, target, kind string }
	var order []bucket
	buckets := map[bucket][]string{}
	for _, e := range links {
		n := byID[e.Source]
		if grouped[n.ID] || n.Test {
			continue
		}
		key := bucket{n.Component + "/" + n.Package, e.Target, e.Kind}
		if _, seen := buckets[key]; !seen {
			order = append(order, key)
		}
		buckets[key] = append(buckets[key], n.ID)
	}
	for _, key := range order {
		var fresh []string
		for _, m := range buckets[key] {
			if !grouped[m] {
				fresh = append(fresh, m)
			}
		}
		if len(fresh) < groupAt {
			continue
		}
		name := byID[key.target].Name
		id := "of-" + notIDText.ReplaceAllString(strings.ToLower(name), "-") + "-" + strconv.Itoa(len(groups)+1)
		word := tr.DiagramInheriting
		if key.kind == "realize" {
			word = tr.DiagramConforming
		}
		add(id, utils.ResolvePlaceholderString(word, map[string]string{"name": name}), "", fresh, key.target, key.kind)
	}
	return groups
}

func regroupEdges(tr *i18n.TranslationSet, edges []models.Edge, groups []group, byID map[string]*Node) []models.Edge {
	if len(groups) == 0 {
		return edges
	}
	var out []models.Edge
	for _, e := range edges {
		if !slices.ContainsFunc(groups, func(g group) bool {
			return slices.Contains(g.members, e.Source) && e.Target == g.target && e.Kind == g.kind
		}) {
			out = append(out, e)
		}
	}
	for _, g := range groups {
		if g.target == "" || byID[g.target] == nil {
			continue
		}
		label := tr.DiagramEachInherits
		if g.kind == "realize" {
			label = tr.DiagramEachConforms
		}
		out = append(out, models.Edge{Source: "group:" + g.id, Target: g.target, Kind: g.kind, Label: label})
	}
	return out
}

func applyLinks(edges []models.Edge, links []components.Link, nodes []*Node, groups []group) []models.Edge {
	find := func(name string) string {
		if id, isGroup := strings.CutPrefix(name, "group:"); isGroup {
			if slices.ContainsFunc(groups, func(g group) bool { return g.id == id }) {
				return name
			}
			return ""
		}
		for _, n := range nodes {
			if n.ID == name {
				return n.ID
			}
		}
		var named []*Node
		for _, n := range nodes {
			if n.Name == name {
				named = append(named, n)
			}
		}
		if len(named) == 1 {
			return named[0].ID
		}
		return ""
	}
	out := append([]models.Edge{}, edges...)
	for _, l := range links {
		source, target := find(l.From), find(l.To)
		if source == "" || target == "" {
			continue
		}
		if i := slices.IndexFunc(out, func(e models.Edge) bool { return e.Source == source && e.Target == target }); i >= 0 {
			out[i].Label, out[i].Key = l.Label, true
			switch {
			case l.Kind == "has":
				out[i].Kind = "has"
			case out[i].Kind == "test":
				out[i].Kind = "use"
			}
			continue
		}
		out = append(out, models.Edge{Source: source, Target: target, Kind: l.Kind, Label: l.Label, Key: true})
	}
	return out
}

func summarize(edges []models.Edge, componentOf func(string) string, labels []components.Summary, componentCount int) []models.ComponentLinks {
	type pair struct{ source, target string }
	var order []pair
	links := map[pair][][2]string{}
	for _, e := range edges {
		key := pair{componentOf(e.Source), componentOf(e.Target)}
		if key.source == key.target {
			continue
		}
		if _, seen := links[key]; !seen {
			order = append(order, key)
		}
		links[key] = append(links[key], [2]string{e.Source, e.Target})
	}
	named := map[pair]string{}
	for _, l := range labels {
		named[pair{l.From, l.To}] = l.Label
	}
	out := []models.ComponentLinks{}
	for _, key := range order {
		_, labelled := named[key]
		if len(links[key]) >= 2 || labelled || componentCount <= 4 {
			out = append(out, models.ComponentLinks{Source: key.source, Target: key.target, Count: len(links[key]), Label: named[key], Links: links[key]})
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		if c := utils.LocaleCompare(out[a].Source, out[b].Source); c != 0 {
			return c < 0
		}
		return utils.LocaleCompare(out[a].Target, out[b].Target) < 0
	})
	return out
}

func stereotypeOf(tr *i18n.TranslationSet, n *Node, names map[string]bool) string {
	switch n.Kind {
	case "artifact":
		return tr.DiagramFile
	case "extension":
		if len(n.Extends) == 0 {
			return tr.DiagramExtension
		}
		return utils.ResolvePlaceholderString(tr.DiagramExtensionOf, map[string]string{"names": strings.Join(n.Extends, ", ")})
	}
	var supers []string
	for _, s := range n.Inherits {
		if !names[s] {
			supers = append(supers, s)
		}
	}
	var parts []string
	switch {
	case n.Label != "":
		parts = append(parts, "«"+n.Label+"»")
	case n.Kind != "class":
		parts = append(parts, "«"+n.Kind+"»")
	}
	if len(supers) > 0 {
		tail := ": " + strings.Join(supers[:min(2, len(supers))], ", ")
		if len(supers) > 2 {
			tail += ", …"
		}
		parts = append(parts, tail)
	}
	return strings.Join(parts, " ")
}

func colorComponents(placed []placement.Component) []placement.Component {
	used := map[string]bool{}
	for _, c := range placed {
		if c.Color != "" {
			used[c.Color] = true
		}
	}
	var free []string
	for _, c := range components.Colors {
		if c != "grey" && !used[c] {
			free = append(free, c)
		}
	}
	k := 0
	out := make([]placement.Component, len(placed))
	for i, c := range placed {
		out[i] = c
		switch {
		case c.Color != "":
		case c.ID == placement.ContextComponent || c.Context:
			out[i].Color = "grey"
		case len(free) > 0:
			out[i].Color = free[k%len(free)]
			k++
		default:
			out[i].Color = components.Colors[k%(len(components.Colors)-1)]
			k++
		}
	}
	return out
}

func orNone[T any](list []T) []T {
	if list == nil {
		return []T{}
	}
	return list
}
