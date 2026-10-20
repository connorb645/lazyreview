package claude

// Asked is one question for Claude Code: the instructions added to its own,
// the message itself, the folder it works in, the session its conversation
// is saved as, or goes on from when Resume, and the settings that outrank
// the user's own.
type Asked struct {
	System   string
	Prompt   string
	Folder   string
	Session  string
	Resume   bool
	Settings any
}
