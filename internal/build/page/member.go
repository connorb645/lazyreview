package page

// Member is one head a page reads: a pull request, with its number; a
// branch, with none; or a branch of a stack of branches, numbered by its
// place in the stack from 1 at the bottom and named in Branch.
type Member struct {
	Pull   int
	Branch string
	Head   string
}
