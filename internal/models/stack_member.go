package models

// StackMember is one member of a stack, with the size of its own diff: a
// pull request, by its number, or a branch, numbered by its place in the
// stack from 1 at the bottom and named in Branch.
type StackMember struct {
	Pull   int    `json:"pull"`
	Branch string `json:"branch,omitempty"`
	Head   string `json:"head"`
	Files  int    `json:"files"`
	Adds   int    `json:"adds"`
	Dels   int    `json:"dels"`
}
