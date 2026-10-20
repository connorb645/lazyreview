package stack

// HunkLines is one hunk of a file's diff with its line numbers on both
// sides: the new-side numbers it adds, the old-side numbers it removes, the
// new-side position each removed line was removed at, and its unchanged
// lines both ways.
type HunkLines struct {
	OldStart  int
	OldLength int
	NewStart  int
	NewLength int
	Added     map[int]bool
	Removed   map[int]bool
	RemovedAt map[int]int
	NewToOld  map[int]int
	OldToNew  map[int]int
}
