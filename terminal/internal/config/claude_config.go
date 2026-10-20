package config

// ClaudeConfig is the Claude Code a question is asked of: its command, and
// the model it answers with, or none for the user's own default.
type ClaudeConfig struct {
	Command string `json:"command"`
	Model   string `json:"model"`
}
