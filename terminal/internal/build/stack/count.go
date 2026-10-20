package stack

// Count is the size of one pull request's own diff: its files, its commits
// and the lines it adds and removes.
type Count struct {
	Pull    int
	Files   int
	Commits int
	Adds    int
	Dels    int
}
