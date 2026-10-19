package config

// KeybindingUniversalConfig is the keys every place takes, the home screen's
// and a review's alike.
type KeybindingUniversalConfig struct {
	Command []string `json:"command"`
	Keys    []string `json:"keys"`
	Review  []string `json:"review"`
	GitHub  []string `json:"github"`
	Close   []string `json:"close"`
	Quit    []string `json:"quit"`
	Refresh []string `json:"refresh"`
	Bigger  []string `json:"bigger"`
	Smaller []string `json:"smaller"`
}
