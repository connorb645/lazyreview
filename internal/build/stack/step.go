package stack

// Step is what ties one pull request of a stack to the stack's combined
// diff: Own is the diff the pull request shows, from Start, where it starts,
// to its head; FromTop is from its head to the stack's top, which says where
// a line of the top sits at its head; FromBase is from the combined base to
// Start, which says where a line of the base sits there.
type Step struct {
	Pull     int
	Start    string
	Own      StepDiff
	FromTop  StepDiff
	FromBase StepDiff
}
