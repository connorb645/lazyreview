package config

// KeybindingNotesPanelConfig is the Notes panel's own keys.
type KeybindingNotesPanelConfig struct {
	Edit       []string `json:"edit"`
	Drop       []string `json:"drop"`
	ToggleFold []string `json:"toggleFold"`
	Post       []string `json:"post"`
	Stop       []string `json:"stop"`
}
