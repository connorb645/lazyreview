package server

// Message is one turn of a conversation: a question from you, or Claude's
// answer, or why there is none.
type Message struct {
	From  string `json:"from"`
	Text  string `json:"text"`
	Error string `json:"error"`
	At    string `json:"at"`
}
