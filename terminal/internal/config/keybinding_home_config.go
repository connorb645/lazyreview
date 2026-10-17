package config

// KeybindingHomeConfig is the home screen's keys. JumpToRepository holds the
// keys of the repositories 1 to 9, each by its place, "" where one has none.
type KeybindingHomeConfig struct {
	Down             []string `json:"down"`
	Up               []string `json:"up"`
	Top              []string `json:"top"`
	Bottom           []string `json:"bottom"`
	Open             []string `json:"open"`
	PreviousPanel    []string `json:"previousPanel"`
	NextPanel        []string `json:"nextPanel"`
	JumpToRepository []string `json:"jumpToRepository"`
}
