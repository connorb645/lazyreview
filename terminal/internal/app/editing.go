package app

import "review-desk/terminal/internal/server"

// editing is a text being written in the editor: a comment, a question to
// Claude, a follow-up in a conversation or an edit of a queued comment,
// with what it is on, the words that say so, and the key its draft is kept by.
type editing struct {
	purpose      string
	at           server.Lines
	pull         int
	comment      int
	version      string
	conversation int
	text         string
	about        []string
	key          string
}
