package layout

import (
	"slices"
	"sort"
)

const orderingSweeps = 4

func layersOf(part []int, below [][]int) [][]int {
	state := map[int]int{}
	kept := map[int][]int{}
	var finished []int
	var visit func(v int)
	visit = func(v int) {
		state[v] = 1
		for _, w := range below[v] {
			if state[w] == 1 || slices.Contains(kept[v], w) {
				continue
			}
			kept[v] = append(kept[v], w)
			if state[w] == 0 {
				visit(w)
			}
		}
		state[v] = 2
		finished = append(finished, v)
	}
	for _, v := range part {
		if state[v] == 0 {
			visit(v)
		}
	}
	layer := map[int]int{}
	incoming := map[int]int{}
	for i := len(finished) - 1; i >= 0; i-- {
		v := finished[i]
		for _, w := range kept[v] {
			layer[w] = max(layer[w], layer[v]+1)
			incoming[w]++
		}
	}
	for _, v := range finished {
		if incoming[v] == 0 && len(kept[v]) > 0 {
			lowest := layer[kept[v][0]]
			for _, w := range kept[v][1:] {
				lowest = min(lowest, layer[w])
			}
			layer[v] = lowest - 1
		}
	}
	least := layer[part[0]]
	deepest := 0
	for _, v := range part {
		least = min(least, layer[v])
	}
	for _, v := range part {
		layer[v] -= least
		deepest = max(deepest, layer[v])
	}
	rows := make([][]int, deepest+1)
	for _, v := range part {
		rows[layer[v]] = append(rows[layer[v]], v)
	}
	neighbours := map[int][]int{}
	for v, ws := range kept {
		for _, w := range ws {
			neighbours[v] = append(neighbours[v], w)
			neighbours[w] = append(neighbours[w], v)
		}
	}
	for range orderingSweeps {
		for k := 1; k < len(rows); k++ {
			reorder(rows[k], rows[k-1], neighbours)
		}
		for k := len(rows) - 2; k >= 0; k-- {
			reorder(rows[k], rows[k+1], neighbours)
		}
	}
	return rows
}

func reorder(row, beside []int, neighbours map[int][]int) {
	place := map[int]int{}
	for i, v := range beside {
		place[v] = i
	}
	key := map[int]float64{}
	for i, v := range row {
		sum, count := 0.0, 0
		for _, w := range neighbours[v] {
			if at, ok := place[w]; ok {
				sum += float64(at)
				count++
			}
		}
		if count == 0 {
			key[v] = float64(i) * float64(max(len(beside), 1)) / float64(max(len(row), 1))
			continue
		}
		key[v] = sum / float64(count)
	}
	sort.SliceStable(row, func(a, b int) bool { return key[row[a]] < key[row[b]] })
}
