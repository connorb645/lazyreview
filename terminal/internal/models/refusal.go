package models

// Refusal is lazyreview saying no, in its own words, such as why a comment
// cannot be queued, from the server or from the terminal app's own reading
// of a review. Status is the server's answer's status, when the server
// said it.
type Refusal struct {
	Status  int
	Message string
}

// Refused is the refusal message words.
func Refused(message string) Refusal {
	return Refusal{Message: message}
}

// Error is the refusal's own words.
func (r Refusal) Error() string {
	return r.Message
}
