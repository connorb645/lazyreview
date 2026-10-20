package config

// KeybindingDiagramConfig is the keys of the diagram in the main panel.
type KeybindingDiagramConfig struct {
	Left          []string `json:"left"`
	Down          []string `json:"down"`
	Up            []string `json:"up"`
	Right         []string `json:"right"`
	NextBox       []string `json:"nextBox"`
	PreviousBox   []string `json:"previousBox"`
	Top           []string `json:"top"`
	Bottom        []string `json:"bottom"`
	Open          []string `json:"open"`
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
