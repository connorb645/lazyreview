package config

// KeybindingHomeConfig is the home screen's keys. JumpToRepository holds the
// keys of the repositories 1 to 9, each by its place, "" where one has none.
// HideRepository hides the repository selected, or shows a hidden one again;
// ShowHidden shows the hidden repositories, or hides them again.
type KeybindingHomeConfig struct {
	Down             []string `json:"down"`
	Up               []string `json:"up"`
	Top              []string `json:"top"`
	Bottom           []string `json:"bottom"`
	HalfDown         []string `json:"halfDown"`
	HalfUp           []string `json:"halfUp"`
	PageDown         []string `json:"pageDown"`
	PageUp           []string `json:"pageUp"`
	Open             []string `json:"open"`
	PreviousPanel    []string `json:"previousPanel"`
	NextPanel        []string `json:"nextPanel"`
	JumpToRepository []string `json:"jumpToRepository"`
	HideRepository   []string `json:"hideRepository"`
	ShowHidden       []string `json:"showHidden"`
}
