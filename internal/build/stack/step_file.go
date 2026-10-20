package stack

// StepFile is one file of a diff between two commits of a stack: its path,
// empty when the diff deletes it, its path before, empty when the diff adds
// it, its status and its hunks.
type StepFile struct {
	Path    string
	OldPath string
	Status  string
	Hunks   []HunkLines
}
