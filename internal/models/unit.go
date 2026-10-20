package models

// Unit is what one review takes: a pull request alone, or a stack, bottom
// first. Review is the review holding every member, or empty.
type Unit struct {
	Review  string     `json:"review"`
	Members []OpenPull `json:"members"`
}
