package layout

import (
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

const (
	canvasAspect = 1.45
	areaPad      = 90
	marginTop    = 170
	marginSide   = 80
	linkedApart  = 420
	looseApart   = 300
)

func layoutCanvas(tr *i18n.TranslationSet, m models.Model, results map[string]laid) drawn {
	pairs := summaryPairs(tr, m.Summary)
	areaOf := map[string]string{}
	for _, a := range m.Areas {
		for _, id := range a.Components {
			areaOf[id] = a.ID
		}
	}
	blockOf := func(id string) string {
		if a, ok := areaOf[id]; ok {
			return "area:" + a
		}
		return "loose"
	}
	var blocks []block
	for i := range m.Areas {
		blocks = append(blocks, block{id: "area:" + m.Areas[i].ID, area: &m.Areas[i], components: m.Areas[i].Components})
	}
	var loose []string
	for _, c := range m.Components {
		if _, ok := areaOf[c.ID]; !ok {
			loose = append(loose, c.ID)
		}
	}
	if len(loose) > 0 {
		blocks = append(blocks, block{id: "loose", components: loose})
	}
	var cross []summaryPair
	for _, p := range pairs {
		if blockOf(p.fwd.Source) != blockOf(p.fwd.Target) {
			cross = append(cross, p)
		}
	}
	for i := range blocks {
		b := &blocks[i]
		for _, p := range pairs {
			if blockOf(p.fwd.Source) == b.id && blockOf(p.fwd.Target) == b.id {
				b.inner = append(b.inner, p)
			}
		}
		layoutBlock(b, results)
	}
	origins := placeBlocks(blocks, cross, blockOf)
	out := drawn{components: map[string]box{}, boxes: map[string]box{}, arrows: []models.PackageArrow{}, summaries: []models.ComponentArrow{}}
	for _, b := range blocks {
		origin := origins[b.id]
		for _, id := range b.components {
			r, ok := results[id]
			if !ok {
				continue
			}
			x, y := origin.x+b.at[id].x, origin.y+b.at[id].y
			out.components[id] = box{x: x, y: y, w: r.w, h: r.h}
			for bid, inner := range r.boxes {
				out.boxes[bid] = box{x: inner.x + x, y: inner.y + y, w: inner.w, h: inner.h}
			}
			for _, a := range r.arrows {
				a.Component = id
				out.arrows = append(out.arrows, a)
			}
		}
		for _, p := range b.inner {
			out.summaries = append(out.summaries, p.arrow())
		}
	}
	for _, p := range cross {
		out.summaries = append(out.summaries, p.arrow())
	}
	return out
}

func layoutBlock(b *block, results map[string]laid) {
	var items []item
	for _, id := range b.components {
		if r, ok := results[id]; ok {
			items = append(items, item{id: id, w: r.w, h: r.h})
		}
	}
	pad := padding{}
	if b.area != nil {
		pad = padding{top: areaHead, side: areaPad, bottom: areaPad}
	}
	packed := shelfPack(items, pad, canvasAspect)
	b.w, b.h = packed.w, packed.h
	if b.area != nil {
		b.w = max(b.w, measure(b.area.Title, "areaTitle")+140)
		if b.area.Description != "" {
			b.w = max(b.w, measure(b.area.Description, "areaTag")+140)
		}
	}
	dx := (b.w - packed.w) / 2
	b.at = map[string]box{}
	for id, at := range packed.at {
		at.x += dx
		b.at[id] = at
	}
}

func placeBlocks(blocks []block, cross []summaryPair, blockOf func(string) string) map[string]box {
	ordered := blockOrder(blocks, cross, blockOf)
	linked := map[string]bool{}
	for _, p := range cross {
		a, b := blockOf(p.fwd.Source), blockOf(p.fwd.Target)
		linked[a+">"+b], linked[b+">"+a] = true, true
	}
	widest := 0.0
	for _, b := range ordered {
		widest = max(widest, b.w)
	}
	at := map[string]box{}
	y := float64(marginTop)
	for k, b := range ordered {
		switch {
		case k == 0:
		case linked[ordered[k-1].id+">"+b.id]:
			y += linkedApart
		default:
			y += looseApart
		}
		at[b.id] = box{x: marginSide + (widest-b.w)/2, y: y}
		y += b.h
	}
	return at
}

func blockOrder(blocks []block, cross []summaryPair, blockOf func(string) string) []block {
	incoming := map[string]int{}
	next := map[string][]string{}
	for _, p := range cross {
		a, b := blockOf(p.fwd.Source), blockOf(p.fwd.Target)
		next[a] = append(next[a], b)
		incoming[b]++
	}
	done := map[string]bool{}
	var out []block
	for len(out) < len(blocks) {
		pick := -1
		for i, b := range blocks {
			if !done[b.id] && incoming[b.id] == 0 {
				pick = i
				break
			}
		}
		if pick < 0 {
			for i, b := range blocks {
				if !done[b.id] {
					pick = i
					break
				}
			}
		}
		b := blocks[pick]
		done[b.id] = true
		out = append(out, b)
		for _, to := range next[b.id] {
			incoming[to]--
		}
	}
	return out
}

func summaryPairs(tr *i18n.TranslationSet, summary []models.ComponentLinks) []summaryPair {
	byKey := map[string]models.ComponentLinks{}
	for _, s := range summary {
		byKey[s.Source+">"+s.Target] = s
	}
	words := func(x models.ComponentLinks, arrow string) string {
		label := ""
		if x.Label != "" {
			label = utils.ResolvePlaceholderString(tr.DiagramSummaryLabel, map[string]string{"label": x.Label})
		}
		return arrow + " " + label + utils.Plural(x.Count, tr.LinkNoun)
	}
	way := func(x models.ComponentLinks, arrow string) models.ArrowWay {
		return models.ArrowWay{Arrow: arrow, Label: x.Label, Links: x.Links}
	}
	var pairs []summaryPair
	for _, s := range summary {
		rev, ok := byKey[s.Target+">"+s.Source]
		if ok && (rev.Count > s.Count || rev.Count == s.Count && rev.Source < s.Source) {
			continue
		}
		p := summaryPair{fwd: s, lines: []string{words(s, "→")}, ways: []models.ArrowWay{way(s, "→")}}
		if ok {
			p.rev = &rev
			p.lines = append(p.lines, words(rev, "←"))
			p.ways = append(p.ways, way(rev, "←"))
		}
		pairs = append(pairs, p)
	}
	return pairs
}
