package types

// PullsLoaded is a stack, or a lone pull request, loaded from GitHub into
// a review, with the server's notes, or why not.
type PullsLoaded struct {
	Unit  string
	ID    string
	Notes []string
	Err   error
}
