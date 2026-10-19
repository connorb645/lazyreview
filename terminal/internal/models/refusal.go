package models

// Refusal is the server saying no, in its own words, such as why a comment
// cannot be queued.
type Refusal struct {
	Status  int
	Message string
}

// Error is the server's own words.
func (r Refusal) Error() string {
	return r.Message
}
