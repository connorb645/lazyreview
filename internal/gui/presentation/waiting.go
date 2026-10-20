package presentation

import "github.com/connorb645/lazyreview/internal/models"

// Waiting says whether a conversation waits for Claude: its last message is
// a question.
func Waiting(c models.Conversation) bool {
	return len(c.Messages) > 0 && c.Messages[len(c.Messages)-1].From == "you"
}
