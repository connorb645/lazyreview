package types

import "github.com/connorb645/lazyreview/internal/models"

// Editing is a comment, a question, a follow-up, an edit or a drop being
// made: what for, on which lines, of which pull request, the queued comment
// and its version, the conversation, its text and whether that is Claude's
// answer, the lines that tell what it is about, and the key its draft is
// kept under.
type Editing struct {
	Purpose      string
	At           models.Lines
	Pull         int
	Comment      int
	Version      string
	Conversation int
	Text         string
	Answer       bool
	About        []string
	Key          string
}
