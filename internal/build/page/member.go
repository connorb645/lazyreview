package page

// Member is one head a page reads: a pull request, with its number, or a
// branch, with none.
type Member struct {
	Pull int
	Head string
}
