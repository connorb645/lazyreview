package stack

// Step is what ties one member of a stack to the stack's combined diff,
// with the member's number and, for a branch, its name: Own is the diff the
// member shows, from Start, where it starts, to its head; FromTop is from
// its head to the stack's top, which says where a line of the top sits at
// its head; FromBase is from the combined base to Start, which says where a
// line of the base sits there.
type Step struct {
	Pull     int
	Branch   string
	Start    string
	Own      StepDiff
	FromTop  StepDiff
	FromBase StepDiff
}
