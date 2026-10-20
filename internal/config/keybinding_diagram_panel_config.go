package config

// KeybindingDiagramPanelConfig is the Diagram panel's own keys, on top of
// the panels'.
type KeybindingDiagramPanelConfig struct {
	NextBox       []string `json:"nextBox"`
	PreviousBox   []string `json:"previousBox"`
	Back          []string `json:"back"`
	Follow        []string `json:"follow"`
	References    []string `json:"references"`
	Search        []string `json:"search"`
	SearchBack    []string `json:"searchBack"`
	NextMatch     []string `json:"nextMatch"`
	PreviousMatch []string `json:"previousMatch"`
	Labels        []string `json:"labels"`
	Links         []string `json:"links"`
	Properties    []string `json:"properties"`
	BoxDiff       []string `json:"boxDiff"`
	AllLinks      []string `json:"allLinks"`
}
