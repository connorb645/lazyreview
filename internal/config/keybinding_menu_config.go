package config

// KeybindingMenuConfig is a menu's keys.
type KeybindingMenuConfig struct {
	Down     []string `json:"down"`
	Up       []string `json:"up"`
	Top      []string `json:"top"`
	Bottom   []string `json:"bottom"`
	HalfDown []string `json:"halfDown"`
	HalfUp   []string `json:"halfUp"`
	PageDown []string `json:"pageDown"`
	PageUp   []string `json:"pageUp"`
	Open     []string `json:"open"`
	Close    []string `json:"close"`
}
