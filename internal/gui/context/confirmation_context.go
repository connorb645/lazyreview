package context

import "github.com/connorb645/lazyreview/internal/models"

// ConfirmationContext asks whether to drop a queued comment, of a pull
// request, at the version shown, or to delete a conversation with Claude,
// as shown.
type ConfirmationContext struct {
	Comment      int
	Pull         int
	Version      string
	Conversation *models.Conversation
}
