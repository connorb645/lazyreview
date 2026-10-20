package models

// ReviewMember is one pull request of a review as review.json keeps it:
// where its own diff starts, what GitHub said of it, its colour, its queued
// comments and the reviews sent from it.
type ReviewMember struct {
	Pull        int            `json:"pull"`
	Head        string         `json:"head"`
	Start       string         `json:"start"`
	Title       string         `json:"title"`
	URL         string         `json:"url"`
	Author      string         `json:"author"`
	Base        string         `json:"base"`
	Description string         `json:"description"`
	State       string         `json:"state"`
	EndedAt     *string        `json:"ended_at"`
	Color       int            `json:"color"`
	Comments    []Comment      `json:"comments"`
	Posted      []PostedReview `json:"posted"`
}

// MarshalJSON writes the member with its comments and posted reviews as
// lists, empty when there are none.
func (m ReviewMember) MarshalJSON() ([]byte, error) {
	type plain ReviewMember
	p := plain(m)
	p.Comments, p.Posted = orNone(p.Comments), orNone(p.Posted)
	return plainJSON(p)
}
