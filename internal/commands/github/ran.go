package github

// Ran is what one run of gh did: its exit code, its output and its error
// output, and whether it ran out of time.
type Ran struct {
	Code     int
	Stdout   string
	Stderr   string
	TimedOut bool
}
