package server

// Written is a written review attached to a review. Stale says it read
// another head than the review now shows.
type Written struct {
	Title    string    `json:"title"`
	Head     string    `json:"head"`
	Sections []Section `json:"sections"`
	Stale    bool      `json:"stale"`
}
