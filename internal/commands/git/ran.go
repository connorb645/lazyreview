package git

// Ran is what one run of git did: its exit code and its output.
type Ran struct {
	Code   int
	Stdout string
	Stderr string
}
