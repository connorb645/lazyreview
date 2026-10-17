package config

// ThemeConfig is the config's gui.theme: the colours for a dark terminal
// and those for a light one.
type ThemeConfig struct {
	Dark  Colours `json:"dark"`
	Light Colours `json:"light"`
}
