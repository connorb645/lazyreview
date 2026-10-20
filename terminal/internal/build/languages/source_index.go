package languages

type sourceIndex struct {
	starts []int
	depths []int32
	closes []int32
}

func indexSource(masked string) sourceIndex {
	n := len(masked)
	x := sourceIndex{starts: []int{0}, depths: make([]int32, n+1), closes: make([]int32, n)}
	for k := range x.closes {
		x.closes[k] = -1
	}
	var stack []int
	var depth int32
	for k := 0; k < n; k++ {
		x.depths[k] = depth
		switch masked[k] {
		case '\n':
			x.starts = append(x.starts, k+1)
		case '{':
			stack = append(stack, k)
			depth++
		case '}':
			if len(stack) > 0 {
				x.closes[stack[len(stack)-1]] = int32(k)
				stack = stack[:len(stack)-1]
			}
			if depth > 0 {
				depth--
			}
		}
	}
	x.depths[n] = depth
	return x
}

func (x sourceIndex) line(offset int) int {
	return lineOf(x.starts, offset) + 1
}

func (x sourceIndex) depth(offset int) int {
	return int(x.depths[min(max(offset, 0), len(x.closes))])
}

func (x sourceIndex) closing(offset int) int {
	if offset >= 0 && offset < len(x.closes) && x.closes[offset] >= 0 {
		return int(x.closes[offset])
	}
	return len(x.closes)
}

func lineOf(starts []int, offset int) int {
	lo, hi := 0, len(starts)-1
	for lo < hi {
		mid := (lo + hi + 1) >> 1
		if starts[mid] <= offset {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}
