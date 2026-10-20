package land

// Ran is what one run of git did: whether it succeeded, and what it
// printed, trimmed.
type Ran struct {
	OK  bool
	Out string
	Err string
}
