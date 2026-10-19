package presentation

import "github.com/connorb645/lazyreview/terminal/internal/models"

// Note is something said on a file's lines: a queued comment, a
// conversation with Claude, or a thread on GitHub.
type Note struct {
	Kind         string
	Key          string
	At           *models.Lines
	Pull         int
	Comment      *models.Comment
	Conversation *models.Conversation
	Thread       *models.GitHubThread
}
