package ask

// Decision is the guard's answer to one tool call: allowed, or refused
// with why.
type Decision struct {
	Allow  bool
	Refuse string
}
