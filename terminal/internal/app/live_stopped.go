package app

// liveStopped says the app stopped hearing a review's changes, and why.
type liveStopped struct {
	id  string
	err error
}
