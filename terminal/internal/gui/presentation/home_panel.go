package presentation

// HomePanel is one panel of the home screen: a repository's pull requests,
// or the accounts GitHub could not read, or a note when there is no list.
// Number is its key from 1 to 9, or 0; Repo its place in the list, or -1.
type HomePanel struct {
	Title   string
	Number  int
	Repo    int
	Waiting int
	Rows    []SideRow
}
