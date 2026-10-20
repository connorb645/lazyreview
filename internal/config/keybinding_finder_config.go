package config

// KeybindingFinderConfig is the finder's keys.
type KeybindingFinderConfig struct {
	Down  []string `json:"down"`
	Up    []string `json:"up"`
	Open  []string `json:"open"`
	Close []string `json:"close"`
	Clear []string `json:"clear"`
}
