package models

// BranchMember is one branch of a stack of branches as review.json keeps
// it: its name, its head, where its own diff starts and its colour.
type BranchMember struct {
	Branch string `json:"branch"`
	Head   string `json:"head"`
	Start  string `json:"start"`
	Color  int    `json:"color"`
}
