package app

import "review-desk/terminal/internal/server"

// note is something drawn under the line it is on in the diff: a queued
// comment, a conversation with Claude, or a thread already on GitHub.
type note struct {
	kind         string
	key          string
	at           *server.Lines
	pull         int
	comment      *server.Comment
	conversation *server.Conversation
	thread       *server.GitHubThread
}
