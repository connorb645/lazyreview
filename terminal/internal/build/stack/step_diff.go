package stack

// StepDiff is a diff between two commits of a stack, its files by new path
// and by old path. A diff between a commit and itself is empty.
type StepDiff struct {
	ByNew map[string]StepFile
	ByOld map[string]StepFile
}
