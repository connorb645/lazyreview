package config

// KeybindingConfirmationConfig is the keys of the question whether to drop a
// queued comment, or to delete a conversation with Claude.
type KeybindingConfirmationConfig struct {
	Confirm []string `json:"confirm"`
}
