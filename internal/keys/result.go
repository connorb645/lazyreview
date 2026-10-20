package keys

// Result is what one key press made of a sequence: a complete one names its
// action and the count typed before it, 1 when none was typed. Waiting
// means the keys so far begin a sequence, and more must come. With neither,
// the keys meant nothing, and were let go.
type Result struct {
	Action  string
	Count   int
	Counted bool
	Waiting bool
}
