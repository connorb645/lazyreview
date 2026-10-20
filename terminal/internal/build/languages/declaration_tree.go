package languages

import "sort"

func nestDeclarations(found []Declaration, isTop func(Declaration) bool) []Declaration {
	sorted := append([]Declaration{}, found...)
	sort.SliceStable(sorted, func(a, b int) bool {
		if sorted[a].Span[0] != sorted[b].Span[0] {
			return sorted[a].Span[0] < sorted[b].Span[0]
		}
		return sorted[a].Span[1] > sorted[b].Span[1]
	})
	var open []Declaration
	for i, d := range sorted {
		for len(open) > 0 && open[len(open)-1].Span[1] <= d.Span[0] {
			open = open[:len(open)-1]
		}
		d.Parent, d.Owner = nil, d.Name
		if len(open) > 0 {
			parent := open[len(open)-1].Name
			d.Parent, d.Owner = &parent, open[0].Name
		}
		d.Top = len(open) == 0 && isTop(d)
		open = append(open, d)
		sorted[i] = d
	}
	return sorted
}

func atDepthZero(d Declaration) bool {
	return d.Depth == 0
}
