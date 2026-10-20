package stack

// FilePulls is which pull requests changed a file of a stack's combined
// diff, in stack order, and each one's added and removed lines there.
type FilePulls struct {
	Pulls  []int
	ByPull map[int][2]int
}
