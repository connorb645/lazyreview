package models

// PostedReview is a review sent to GitHub from a pull request's queue: the
// set id the press named, its action, its address on GitHub, when, and the
// comments it sent, as they stood.
type PostedReview struct {
	Set      string    `json:"set"`
	Action   string    `json:"action"`
	Review   string    `json:"review"`
	At       string    `json:"at"`
	Comments []Comment `json:"comments"`
}

// MarshalJSON writes the posted review with its comments as a list, empty
// when it sent none.
func (p PostedReview) MarshalJSON() ([]byte, error) {
	type plain PostedReview
	q := plain(p)
	q.Comments = orNone(q.Comments)
	return plainJSON(q)
}
