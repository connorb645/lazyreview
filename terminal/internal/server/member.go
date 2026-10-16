package server

// Member is one pull request of a review: a single pull request's review has
// one, a stack's several, bottom first.
type Member struct {
	Pull    int     `json:"pull"`
	Head    string  `json:"head"`
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Author  string  `json:"author"`
	Base    string  `json:"base"`
	State   string  `json:"state"`
	EndedAt *string `json:"ended_at"`
	Color   int     `json:"color"`
}
