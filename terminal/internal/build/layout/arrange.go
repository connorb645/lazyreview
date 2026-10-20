package layout

import (
	"math"
	"slices"
)

func arrange(items []item, links [][2]string, a arrangement) placed {
	index := map[string]int{}
	for i, it := range items {
		index[it.id] = i
	}
	below := make([][]int, len(items))
	root := make([]int, len(items))
	for i := range root {
		root[i] = i
	}
	find := func(i int) int {
		for root[i] != i {
			root[i] = root[root[i]]
			i = root[i]
		}
		return i
	}
	linked := make([]bool, len(items))
	for _, l := range links {
		i, iok := index[l[0]]
		j, jok := index[l[1]]
		if !iok || !jok || i == j {
			continue
		}
		below[i] = append(below[i], j)
		linked[i], linked[j] = true, true
		root[find(i)] = find(j)
	}
	if !slices.Contains(linked, true) {
		return tidyRows(items, a.packedGap, a.packedAspect)
	}
	var parts [][]int
	partOf := map[int]int{}
	var singles []item
	for i, it := range items {
		if !linked[i] {
			singles = append(singles, it)
			continue
		}
		r := find(i)
		if _, ok := partOf[r]; !ok {
			partOf[r] = len(parts)
			parts = append(parts, nil)
		}
		parts[partOf[r]] = append(parts[partOf[r]], i)
	}
	var layered [][][]int
	for _, p := range parts {
		layered = append(layered, layersOf(p, below))
	}
	out := shelve(layered, items, a)
	if len(singles) == 0 {
		return out
	}
	rest := tidyRows(singles, a.nodeGap, a.linkedAspect)
	top := out.h + a.partGap
	for id, b := range rest.at {
		b.y += top
		out.at[id] = b
	}
	out.w = max(out.w, rest.w)
	out.h = top + rest.h
	return out
}

func shelve(parts [][][]int, items []item, a arrangement) placed {
	widths := make([]float64, len(parts))
	heights := make([][]float64, len(parts))
	rowWidths := make([][]float64, len(parts))
	widest := 0.0
	for p, rows := range parts {
		for _, row := range rows {
			w, h := 0.0, 0.0
			for k, i := range row {
				if k > 0 {
					w += a.nodeGap
				}
				w += items[i].w
				h = max(h, items[i].h)
			}
			rowWidths[p] = append(rowWidths[p], w)
			heights[p] = append(heights[p], h)
			widths[p] = max(widths[p], w)
		}
		widest = max(widest, widths[p])
	}
	limits := []float64{widest}
	sum := 0.0
	for p, w := range widths {
		if p > 0 {
			sum += a.partGap
		}
		sum += w
		if sum > widest {
			limits = append(limits, sum)
		}
	}
	var best [][]int
	bestFit := math.Inf(1)
	for _, limit := range limits {
		shelves := fillShelves(widths, a.partGap, limit)
		w, h := shelvesSize(shelves, widths, heights, a)
		if fit := math.Abs(math.Log(w / h / a.linkedAspect)); fit < bestFit-1e-9 {
			best, bestFit = shelves, fit
		}
	}
	at := map[string]box{}
	top, wide := 0.0, 0.0
	for s, shelf := range best {
		if s > 0 {
			top += a.partGap
		}
		bands := shelfBands(shelf, heights)
		x := 0.0
		for k, p := range shelf {
			if k > 0 {
				x += a.partGap
			}
			y := top
			for r, row := range parts[p] {
				left := x + (widths[p]-rowWidths[p][r])/2
				for _, i := range row {
					at[items[i].id] = box{x: left, y: y, w: items[i].w, h: items[i].h}
					left += items[i].w + a.nodeGap
				}
				y += bands[r] + a.layerGap
			}
			x += widths[p]
		}
		wide = max(wide, x)
		top += bandsHeight(bands, a.layerGap)
	}
	return placed{w: wide, h: top, at: at}
}

func fillShelves(widths []float64, gap, limit float64) [][]int {
	var shelves [][]int
	x := 0.0
	for p, w := range widths {
		if len(shelves) == 0 || x+gap+w > limit {
			shelves = append(shelves, []int{p})
			x = w
			continue
		}
		shelves[len(shelves)-1] = append(shelves[len(shelves)-1], p)
		x += gap + w
	}
	return shelves
}

func shelvesSize(shelves [][]int, widths []float64, heights [][]float64, a arrangement) (float64, float64) {
	w, h := 0.0, 0.0
	for s, shelf := range shelves {
		x := 0.0
		for k, p := range shelf {
			if k > 0 {
				x += a.partGap
			}
			x += widths[p]
		}
		w = max(w, x)
		if s > 0 {
			h += a.partGap
		}
		h += bandsHeight(shelfBands(shelf, heights), a.layerGap)
	}
	return w, h
}

func shelfBands(shelf []int, heights [][]float64) []float64 {
	var bands []float64
	for _, p := range shelf {
		for r, h := range heights[p] {
			if r == len(bands) {
				bands = append(bands, 0)
			}
			bands[r] = max(bands[r], h)
		}
	}
	return bands
}

func bandsHeight(bands []float64, gap float64) float64 {
	h := 0.0
	for r, b := range bands {
		if r > 0 {
			h += gap
		}
		h += b
	}
	return h
}
