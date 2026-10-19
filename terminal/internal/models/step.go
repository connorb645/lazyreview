package models

// Step is one thing Claude did while it answered, in words, and why it was
// refused when it was.
type Step struct {
	Words   string `json:"words"`
	Refused string `json:"refused"`
}
