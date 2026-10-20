package layout

import (
	"math"
	"slices"

	"github.com/connorb645/lazyreview/internal/utils"
)

const (
	shelfGapX = 460
	shelfGapY = 300
)

func shelfPack(boxes []item, pad padding, aspect float64) placed {
	if len(boxes) == 0 {
		return placed{w: pad.side * 2, h: pad.top + pad.bottom, at: map[string]box{}}
	}
	sorted := slices.Clone(boxes)
	slices.SortStableFunc(sorted, func(a, b item) int {
		if d := b.w*b.h - a.w*a.h; d != 0 {
			if d > 0 {
				return 1
			}
			return -1
		}
		return utils.LocaleCompare(a.id, b.id)
	})
	total, widest := 0.0, 0.0
	for _, b := range sorted {
		total += b.w * b.h
		widest = max(widest, b.w)
	}
	alone := len(sorted) > 2 && sorted[0].w*sorted[0].h > total*0.4
	limits := []float64{widest}
	start := 0
	if alone {
		start = 1
	}
	for i := start; i < len(sorted); i++ {
		width := 0.0
		for j := i; j < len(sorted); j++ {
			if j > i {
				width += shelfGapX
			}
			width += sorted[j].w
			if width >= widest && !slices.Contains(limits, width) {
				limits = append(limits, width)
			}
		}
	}
	var best placed
	bestFit := math.Inf(1)
	for _, limit := range limits {
		p := shelfRows(sorted, pad, limit, alone)
		if fit := math.Abs(math.Log(p.w / p.h / aspect)); fit < bestFit-1e-9 {
			best, bestFit = p, fit
		}
	}
	return best
}

func shelfRows(sorted []item, pad padding, limit float64, alone bool) placed {
	rows := [][]item{{sorted[0]}}
	widths := []float64{sorted[0].w}
	for _, b := range sorted[1:] {
		last := len(rows) - 1
		if (len(rows) > 1 || !alone) && widths[last]+shelfGapX+b.w <= limit {
			rows[last] = append(rows[last], b)
			widths[last] += shelfGapX + b.w
			continue
		}
		rows = append(rows, []item{b})
		widths = append(widths, b.w)
	}
	width := slices.Max(widths)
	at := map[string]box{}
	y := pad.top
	for r, row := range rows {
		x := pad.side + (width-widths[r])/2
		tallest := 0.0
		for _, b := range row {
			at[b.id] = box{x: x, y: y, w: b.w, h: b.h}
			x += b.w + shelfGapX
			tallest = max(tallest, b.h)
		}
		y += tallest + shelfGapY
	}
	return placed{w: width + pad.side*2, h: y - shelfGapY + pad.bottom, at: at}
}
