package app

import "review-desk/terminal/internal/server"

type note struct {
	kind         string
	key          string
	at           *server.Lines
	pull         int
	comment      *server.Comment
	conversation *server.Conversation
	thread       *server.GitHubThread
}
