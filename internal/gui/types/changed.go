package types

import "github.com/connorb645/lazyreview/internal/models"

// Changed is the answer to a change asked for from the editor:
// the state after it, or the conversation Claude answered in, or why not.
type Changed struct {
	E            Editing
	State        *models.State
	Conversation *models.Conversation
	Err          error
}
