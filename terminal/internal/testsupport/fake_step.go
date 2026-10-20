package testsupport

// FakeStep is one tool call the stand-in Claude makes: its tool, its input,
// and the words of the refusal it hears back, if refused.
type FakeStep struct {
	Tool    string         `json:"tool"`
	Input   map[string]any `json:"input,omitempty"`
	Refused string         `json:"refused,omitempty"`
}
