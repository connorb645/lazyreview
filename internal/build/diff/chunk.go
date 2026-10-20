package diff

// Chunk is one file's part of a diff: its path, its path before the change
// when it had one, its status (A, M, D or R), whether it is binary, its own
// text of the diff and its hunks.
type Chunk struct {
	Path    string
	OldPath string
	Status  string
	Binary  bool
	Text    string
	Hunks   []Hunk
}
