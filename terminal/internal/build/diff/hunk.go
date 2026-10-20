package diff

// Hunk is one hunk of a file's diff: where it starts and how many lines it
// spans on each side, the new-side numbers of the lines it adds, and the
// new-side position of each line it removes.
type Hunk struct {
	OldStart  int
	OldLength int
	NewStart  int
	NewLength int
	Added     []int
	RemovedAt []int
}
