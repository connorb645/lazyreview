package testsupport

// FakeAnswer is what the stand-in Claude answers next: its text, the steps
// it takes first, and how many milliseconds it waits before answering,
// until it is stopped.
type FakeAnswer struct {
	Text  string     `json:"text"`
	Steps []FakeStep `json:"steps,omitempty"`
	Wait  int        `json:"wait,omitempty"`
}
