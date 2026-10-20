package models

// Message is one turn of a conversation: a question from you, or Claude's
// answer, or why there is none, with the steps Claude took.
type Message struct {
	From  string `json:"from"`
	Text  string `json:"text,omitempty"`
	Error string `json:"error,omitempty"`
	Steps []Step `json:"steps,omitempty"`
	At    string `json:"at"`
}
