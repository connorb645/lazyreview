package stack

// Member is one member of a stack to read: a pull request, by its number,
// or a branch, numbered by its place in the stack and named in Branch; and
// its head.
type Member struct {
	Pull   int
	Branch string
	Head   string
}
