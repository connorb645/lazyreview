package types

// LiveStopped is the app no longer hearing a review's changes, and why.
type LiveStopped struct {
	ID  string
	Err error
}
