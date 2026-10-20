package queue

// Posting is what a press posted: the words saying what was done and the
// review's address on GitHub.
type Posting struct {
	Message string
	Review  string
}
