package models

// Refusal is lazyreview saying no, in its own words, such as why a comment
// cannot be queued.
type Refusal struct {
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
