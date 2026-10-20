package claude

// Heard is one thing heard while Claude works: a tool call as Claude makes
// it, with its id, its tool and its input, or, for a call already heard by
// its id, why it was refused.
type Heard struct {
	ID      string
	Tool    string
	Input   map[string]any
	Refused string
}
