package model

import (
	"math"
	"slices"
	"sort"
	"strconv"

	"github.com/connorb645/lazyreview/terminal/internal/build/languages"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var (
	outward    = map[string]float64{"inherit": 6, "realize": 6, "has": 4, "extend": 4, "use": 1.5, "test": 0.5}
	inward     = map[string]float64{"inherit": 3, "realize": 3, "has": 3, "extend": 2, "use": 1.5, "test": 0}
	structural = []string{"inherit", "realize", "has", "extend"}
)

// ChooseContext chooses the unchanged types drawn around the change, from
// untouched, the unchanged types inside the changed files, and candidates,
// those from unchanged files. It scores each by how the drawn types link to
// it: conforming to it or inheriting from it counts most, holding it next,
// and naming it least. A link the other way, unchanged code using a drawn
// type, counts too, less so when the drawn type is used everywhere. Without
// options.Enabled, only untouched types a drawn type conforms to, inherits
// from, holds or extends are kept, and no candidate. It returns the kept
// untouched types and the kept candidates.
func ChooseContext(tr *i18n.TranslationSet, drawn, untouched, candidates []*Node, options ContextOptions) ([]*Node, []*Node) {
	originals := append(append([]*Node{}, drawn...), untouched...)
	if options.Enabled {
		originals = append(originals, candidates...)
	}
	pool := make([]*Node, len(originals))
	original := map[string]*Node{}
	drawnIDs := map[string]bool{}
	for i, n := range originals {
		copied := *n
		copied.ID = "n" + strconv.Itoa(i)
		pool[i] = &copied
		original[copied.ID] = n
		if i < len(drawn) {
			drawnIDs[copied.ID] = true
		}
	}

	spread := map[string]int{}
	for _, c := range pool[len(drawn)+len(untouched):] {
		words := languages.Identifiers(c.Code)
		for _, d := range pool[:len(drawn)] {
			if slices.Contains(words, d.Name) {
				spread[d.ID]++
			}
		}
	}
	relations := FindRelations(tr, pool, nil)
	reach := map[string]int{}
	for _, e := range relations {
		if drawnIDs[e.Source] && !drawnIDs[e.Target] {
			reach[e.Source]++
		}
	}
	score := map[string]float64{}
	kinds := map[string]bool{}
	for _, e := range relations {
		switch {
		case drawnIDs[e.Source] && !drawnIDs[e.Target]:
			damping := 1.0
			if e.Kind == "use" {
				damping = 1 / math.Log2(1+float64(reach[e.Source]))
			}
			score[e.Target] += outward[e.Kind] * math.Min(1, damping)
			if slices.Contains(structural, e.Kind) {
				kinds[e.Target] = true
			}
		case !drawnIDs[e.Source] && drawnIDs[e.Target]:
			score[e.Source] += inward[e.Kind] / math.Log2(2+float64(spread[e.Target]))
		}
	}

	unchanged := pool[len(drawn):]
	kept := map[*Node]bool{}
	if !options.Enabled {
		for _, n := range unchanged {
			if kinds[n.ID] {
				kept[original[n.ID]] = true
			}
		}
		return keptOf(untouched, kept), []*Node{}
	}
	var ranked []*Node
	for _, n := range unchanged {
		if !slices.Contains(options.Exclude, n.Name) && !n.Test && (slices.Contains(options.Include, n.Name) || score[n.ID] >= 0.5) {
			ranked = append(ranked, n)
		}
	}
	sort.SliceStable(ranked, func(a, b int) bool {
		x, y := ranked[a], ranked[b]
		if ix, iy := slices.Contains(options.Include, x.Name), slices.Contains(options.Include, y.Name); ix != iy {
			return ix
		}
		if score[x.ID] != score[y.ID] {
			return score[x.ID] > score[y.ID]
		}
		return utils.LocaleCompare(x.Name, y.Name) < 0
	})
	names := map[string]bool{}
	for _, n := range ranked {
		if len(kept) >= options.Max && !slices.Contains(options.Include, n.Name) {
			continue
		}
		key := n.Family + ":" + n.Name
		if names[key] {
			continue
		}
		names[key] = true
		kept[original[n.ID]] = true
	}
	return keptOf(untouched, kept), keptOf(candidates, kept)
}

func keptOf(nodes []*Node, kept map[*Node]bool) []*Node {
	out := []*Node{}
	for _, n := range nodes {
		if kept[n] {
			out = append(out, n)
		}
	}
	return out
}
