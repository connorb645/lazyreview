package models

// Moved says a review moved into a stack's review, which now holds its
// pull requests.
type Moved struct {
	To    string `json:"to"`
	Stack []int  `json:"stack"`
	At    string `json:"at"`
}
