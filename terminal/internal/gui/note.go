package gui

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

type note struct {
	kind         string
	key          string
	at           *server.Lines
	pull         int
	comment      *server.Comment
	conversation *server.Conversation
	thread       *server.GitHubThread
}
