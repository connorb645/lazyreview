package models

// Member is one pull request of a review: a single pull request's review has
// one, a stack's several, bottom first. A stack of branches has a member for
// each branch, numbered by its place in the stack from 1 at the bottom and
// named in Branch, its Base the branch below or the stack's base.
type Member struct {
	Pull    int     `json:"pull"`
	Branch  string  `json:"branch,omitempty"`
	Head    string  `json:"head"`
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Author  string  `json:"author"`
	Base    string  `json:"base"`
	State   string  `json:"state"`
	EndedAt *string `json:"ended_at"`
	Color   int     `json:"color"`
}
