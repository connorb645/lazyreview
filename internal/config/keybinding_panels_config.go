package config

// KeybindingPanelsConfig is the keys every panel on the left takes.
// JumpToBlock holds the keys that focus the Review, Files, Notes and Diagram
// panels, each by its place, "" where one has none.
type KeybindingPanelsConfig struct {
	Down          []string `json:"down"`
	Up            []string `json:"up"`
	Top           []string `json:"top"`
	Bottom        []string `json:"bottom"`
	HalfDown      []string `json:"halfDown"`
	HalfUp        []string `json:"halfUp"`
	PageDown      []string `json:"pageDown"`
	PageUp        []string `json:"pageUp"`
	Open          []string `json:"open"`
	PreviousPanel []string `json:"previousPanel"`
	NextPanel     []string `json:"nextPanel"`
	JumpToBlock   []string `json:"jumpToBlock"`
}
