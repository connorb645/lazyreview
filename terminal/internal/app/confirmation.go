package app

// confirmation is a drop waiting for y: the queued comment and the version
// it was shown at.
type confirmation struct {
	comment int
	pull    int
	version string
}
