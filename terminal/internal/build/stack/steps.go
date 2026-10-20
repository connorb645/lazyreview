package stack

// Steps is a stack as read from its repository, bottom first: each member's
// head commit, its step, its count, and warnings where the stack is out of
// step.
type Steps struct {
	Heads    []string
	Steps    []Step
	Warnings []string
	Counts   []Count
}
