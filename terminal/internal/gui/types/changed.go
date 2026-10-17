package types

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// Changed is the server's answer to a change asked of it from the editor:
// the state after it, or the conversation Claude answered in, or why not.
type Changed struct {
	E            Editing
	State        *server.State
	Conversation *server.Conversation
	Err          error
}
