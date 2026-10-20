package git

// FileCount is how many lines a diff adds and removes in a file, and
// whether the file is binary.
type FileCount struct {
	Adds   int
	Dels   int
	Binary bool
}
