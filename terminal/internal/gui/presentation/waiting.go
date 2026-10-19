package presentation

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// Waiting says whether a conversation waits for Claude: its last message is
// a question.
func Waiting(c server.Conversation) bool {
	return len(c.Messages) > 0 && c.Messages[len(c.Messages)-1].From == "you"
}
