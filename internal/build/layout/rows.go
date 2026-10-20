package layout

import "math"

func tidyRows(items []item, gap, aspect float64) placed {
	if len(items) == 0 {
		return placed{at: map[string]box{}}
	}
	widest := 0.0
	for _, it := range items {
		widest = max(widest, it.w)
	}
	limits := []float64{widest}
	width := 0.0
	for j, it := range items {
		if j > 0 {
			width += gap
		}
		width += it.w
		if width > widest {
			limits = append(limits, width)
		}
	}
	var best placed
	bestFit := math.Inf(1)
	for _, limit := range limits {
		p := fillRows(items, gap, limit)
		if fit := math.Abs(math.Log(p.w / p.h / aspect)); fit < bestFit-1e-9 {
			best, bestFit = p, fit
		}
	}
	return best
}

func fillRows(items []item, gap, limit float64) placed {
	at := map[string]box{}
	x, y, tallest, widest := 0.0, 0.0, 0.0, 0.0
	for _, it := range items {
		if x > 0 && x+it.w > limit {
			y += tallest + gap
			x, tallest = 0, 0
		}
		at[it.id] = box{x: x, y: y, w: it.w, h: it.h}
		widest = max(widest, x+it.w)
		x += it.w + gap
		tallest = max(tallest, it.h)
	}
	return placed{w: widest, h: y + tallest, at: at}
}
