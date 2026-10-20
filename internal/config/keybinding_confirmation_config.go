package config

// KeybindingConfirmationConfig is the keys of the question whether to drop a
// queued comment.
type KeybindingConfirmationConfig struct {
	Confirm []string `json:"confirm"`
}
