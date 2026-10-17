package presentation

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// Note is something said on a file's lines: a queued comment, a
// conversation with Claude, or a thread on GitHub.
type Note struct {
	Kind         string
	Key          string
	At           *server.Lines
	Pull         int
	Comment      *server.Comment
	Conversation *server.Conversation
	Thread       *server.GitHubThread
}
