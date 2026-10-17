package server

// Refusal is the server saying no, in its own words, such as why a comment
// cannot be queued.
type Refusal struct {
	Status  int
	Message string
}

func (r Refusal) Error() string {
	return r.Message
}
