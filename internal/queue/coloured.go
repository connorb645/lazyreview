package queue

const colours = 8

// Coloured is the colours of a stack's members, bottom first, with each 0,
// a member new to the stack, given the first colour no other member has, or
// once every colour is taken, the colour of its place in the stack, going
// round. A member keeps the colour it had.
func Coloured(given []int) []int {
	taken := map[int]bool{}
	for _, c := range given {
		if c != 0 {
			taken[c] = true
		}
	}
	out := make([]int, len(given))
	for i, c := range given {
		if c == 0 {
			c = i%colours + 1
			for free := 1; free <= colours; free++ {
				if !taken[free] {
					c = free
					break
				}
			}
			taken[c] = true
		}
		out[i] = c
	}
	return out
}
