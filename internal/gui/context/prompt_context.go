package context

import "github.com/charmbracelet/bubbles/textinput"

// PromptContext is the line in the bar where a command or a search is
// typed, and what it was opened for: ":" for a command, "/" or "?" for a
// search forward or back.
type PromptContext struct {
	Input textinput.Model
	Kind  string
}
