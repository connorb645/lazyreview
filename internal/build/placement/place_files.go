package placement

import (
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/internal/build/components"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/utils"
)

// ContextComponent is the id of the component of unchanged code no
// context component of the config names.
const ContextComponent = "context"

var notSlug = regexp.MustCompile(`[^a-z0-9]+`)

// PlaceFiles matches every path against the config's patterns, the most
// specific winning and the first in the config breaking a tie. Changed
// files match the ordinary components; with context, the paths are
// unchanged files shown as context, which match only components marked
// context, and whatever matches none goes to one component of unchanged
// code. Files no pattern names are grouped by folder, each group named by
// its folder; a group of the whole repository takes rootTitle.
func PlaceFiles(tr *i18n.TranslationSet, paths []string, config components.Config, context bool, rootTitle string) Placement {
	var declared []components.Component
	for _, c := range config.Components {
		if c.Context == context {
			declared = append(declared, c)
		}
	}
	var rules []rule
	for _, c := range declared {
		for _, p := range c.Packages {
			for _, text := range p.Paths {
				rules = append(rules, rule{pattern: NewPattern(text), component: c.ID, pkg: p.ID})
			}
		}
		for _, text := range c.Paths {
			rules = append(rules, rule{pattern: NewPattern(text), component: c.ID})
		}
	}
	for i := range rules {
		rules[i].order = i
	}

	places := map[string]Place{}
	loose := map[string]bool{}
	var unmatched []string
	for _, path := range paths {
		var hits []rule
		for _, r := range rules {
			if r.pattern.Matches(path) {
				hits = append(hits, r)
			}
		}
		if len(hits) == 0 {
			unmatched = append(unmatched, path)
			continue
		}
		sort.SliceStable(hits, func(a, b int) bool {
			if hits[a].pattern.Specificity != hits[b].pattern.Specificity {
				return hits[a].pattern.Specificity > hits[b].pattern.Specificity
			}
			return hits[a].order < hits[b].order
		})
		places[path] = Place{Component: hits[0].component, Package: hits[0].pkg}
		loose[path] = hits[0].pkg == ""
	}

	var out []Component
	for _, c := range declared {
		var own []string
		for _, p := range paths {
			if place, held := places[p]; held && place.Component == c.ID {
				own = append(own, p)
			}
		}
		if len(own) == 0 {
			continue
		}
		var packages []Package
		var taken []string
		for _, p := range c.Packages {
			if slices.ContainsFunc(own, func(path string) bool { return !loose[path] && places[path].Package == p.ID }) {
				packages = append(packages, Package{ID: p.ID, Title: p.Title, Description: p.Description})
				taken = append(taken, p.ID)
			}
		}
		var unplaced []string
		for _, path := range own {
			if loose[path] {
				unplaced = append(unplaced, path)
			}
		}
		for _, p := range folderPackages(tr, unplaced, taken, 3) {
			for _, path := range p.paths {
				places[path] = Place{Component: c.ID, Package: p.id}
			}
			packages = append(packages, Package{ID: p.id, Title: p.title})
		}
		component := Component{ID: c.ID, Title: c.Title, Description: c.Description, Tag: c.Tag, Color: c.Color, Group: c.Group, Context: c.Context, Declared: true, Packages: packages}
		out = append(out, finish(tr, component, own, places))
	}

	if len(unmatched) > 0 {
		var extra []FolderGroup
		if context {
			extra = []FolderGroup{{Others: true, Paths: unmatched}}
		} else {
			maxGroups := 8
			if len(declared) > 0 {
				maxGroups = 4
			}
			extra = mergeSmall(GroupByFolder(unmatched, Grouping{MaxGroups: maxGroups, SplitAbove: math.Max(8, float64(len(unmatched))/4), Balanced: true}))
		}
		taken := map[string]bool{ContextComponent: true}
		for _, c := range config.Components {
			taken[c.ID] = true
		}
		for _, group := range extra {
			component := Component{Context: context}
			switch {
			case context:
				component.ID, component.Title, component.Description = ContextComponent, tr.DiagramUnchangedCode, tr.DiagramContextDescription
			case group.Others:
				component.ID, component.Title = uniqueID("other", taken), tr.DiagramOtherFiles
			case group.Folder == "":
				component.ID, component.Title = uniqueID("root", taken), rootTitle
			default:
				component.ID, component.Title = uniqueID(slug(group.Folder), taken), titleFor(tr, group.Folder, extra)
			}
			splitAbove := 3.0
			if context {
				splitAbove = 6
			}
			for _, p := range folderPackages(tr, group.Paths, nil, splitAbove) {
				for _, path := range p.paths {
					places[path] = Place{Component: component.ID, Package: p.id}
				}
				component.Packages = append(component.Packages, Package{ID: p.id, Title: p.title})
			}
			out = append(out, finish(tr, component, group.Paths, places))
		}
		sort.SliceStable(out, func(a, b int) bool {
			x, y := out[a], out[b]
			if x.Declared != y.Declared {
				return x.Declared
			}
			if x.Declared {
				return false
			}
			if (x.ID == "other") != (y.ID == "other") {
				return y.ID == "other"
			}
			return x.Size > y.Size
		})
	}
	return Placement{Components: out, Places: places}
}

func finish(tr *i18n.TranslationSet, c Component, paths []string, places map[string]Place) Component {
	for i, p := range c.Packages {
		var own []string
		for _, path := range paths {
			if places[path].Package == p.ID {
				own = append(own, path)
			}
		}
		c.Packages[i].Path = WhereFiles(own)
	}
	if c.Tag == "" {
		c.Tag = WhereFiles(paths)
	}
	if c.Tag == "" {
		c.Tag = tr.DiagramRepositoryRoot
	}
	c.Size = len(paths)
	return c
}

func folderPackages(tr *i18n.TranslationSet, paths, taken []string, splitAbove float64) []folderPackage {
	if len(paths) == 0 {
		return nil
	}
	used := map[string]bool{}
	for _, id := range taken {
		used[id] = true
	}
	base := strings.TrimSuffix(CommonFolder(paths), "/")
	groups := GroupByFolder(paths, Grouping{MaxGroups: 12, SplitAbove: splitAbove, SplitTop: true})
	baseName := tr.DiagramRepositoryRoot
	if base != "" {
		baseName = base[strings.LastIndex(base, "/")+1:]
	}
	out := make([]folderPackage, len(groups))
	for i, g := range groups {
		relative := g.Folder
		switch {
		case g.Folder == base:
			relative = ""
		case base != "":
			relative = g.Folder[len(base)+1:]
		}
		title := relative
		if title == "" {
			title = baseName
			if len(groups) > 1 && base != "" {
				title = utils.ResolvePlaceholderString(tr.DiagramTopLevel, map[string]string{"name": baseName})
			}
		}
		id := relative
		if id == "" {
			id = "top"
		}
		out[i] = folderPackage{id: uniqueID(slug(id), used), title: title, paths: g.Paths}
	}
	return out
}

func mergeSmall(groups []FolderGroup) []FolderGroup {
	if len(groups) <= 3 {
		return groups
	}
	var small, large []FolderGroup
	for _, g := range groups {
		if len(g.Paths) <= 3 || g.Others {
			small = append(small, g)
		}
		if len(g.Paths) > 3 && !g.Others {
			large = append(large, g)
		}
	}
	if len(small) < 2 {
		return groups
	}
	merged := FolderGroup{Others: true}
	for _, g := range small {
		merged.Paths = append(merged.Paths, g.Paths...)
	}
	return append(large, merged)
}

func titleFor(tr *i18n.TranslationSet, folder string, groups []FolderGroup) string {
	if folder == "" {
		return tr.DiagramRepositoryRoot
	}
	segments := strings.Split(folder, "/")
	last := segments[len(segments)-1]
	clash := slices.ContainsFunc(groups, func(g FolderGroup) bool {
		return !g.Others && g.Folder != "" && g.Folder != folder && g.Folder[strings.LastIndex(g.Folder, "/")+1:] == last
	})
	if clash && len(segments) > 1 {
		return strings.Join(segments[len(segments)-2:], "/")
	}
	return last
}

// WhereFiles is where a set of files lives, for a heading: their common
// folder, or when they share none, the folders and root files that hold
// most of them, such as "Tests/AppTests/ · Package.swift · scripts/".
func WhereFiles(paths []string) string {
	common := CommonFolder(paths)
	if common != "" || len(paths) == 0 {
		return common
	}
	counts := map[string]int{}
	var places []string
	for _, p := range paths {
		key := p
		if first, _, nested := strings.Cut(p, "/"); nested {
			key = first + "/"
		}
		if counts[key] == 0 {
			places = append(places, key)
		}
		counts[key]++
	}
	slices.SortStableFunc(places, func(a, b string) int {
		if counts[a] != counts[b] {
			return counts[b] - counts[a]
		}
		return utils.LocaleCompare(a, b)
	})
	where := strings.Join(places[:min(3, len(places))], " · ")
	if len(places) > 3 {
		where += " · …"
	}
	return where
}

// CommonFolder is the deepest folder holding every path, with a trailing
// slash, or "" at the root.
func CommonFolder(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	folders := make([][]string, len(paths))
	for i, p := range paths {
		segments := strings.Split(p, "/")
		folders[i] = segments[:len(segments)-1]
	}
	first := folders[0]
	depth := len(first)
	for _, f := range folders {
		k := 0
		for k < depth && k < len(f) && f[k] == first[k] {
			k++
		}
		depth = k
	}
	if depth == 0 {
		return ""
	}
	return strings.Join(first[:depth], "/") + "/"
}

func slug(text string) string {
	s := strings.Trim(notSlug.ReplaceAllString(strings.ToLower(text), "-"), "-")
	if s == "" {
		return "part"
	}
	return s
}

func uniqueID(base string, taken map[string]bool) string {
	id := base
	for k := 2; taken[id]; k++ {
		id = base + "-" + strconv.Itoa(k)
	}
	taken[id] = true
	return id
}
