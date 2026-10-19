package config

// Colours are the terminal app's colours for a dark or a light terminal,
// one for each role, each written as #rrggbb or as a terminal colour number
// from 0 to 255, as the config's gui.theme gives them. The names are
// lazygit's where the role is the same.
type Colours struct {
	DefaultFgColor                  string            `json:"defaultFgColor"`
	MutedFgColor                    string            `json:"mutedFgColor"`
	FaintFgColor                    string            `json:"faintFgColor"`
	AccentColor                     string            `json:"accentColor"`
	ActiveBorderColor               string            `json:"activeBorderColor"`
	InactiveBorderColor             string            `json:"inactiveBorderColor"`
	AddedColor                      string            `json:"addedColor"`
	RemovedColor                    string            `json:"removedColor"`
	DimAddedColor                   string            `json:"dimAddedColor"`
	DimRemovedColor                 string            `json:"dimRemovedColor"`
	WarningColor                    string            `json:"warningColor"`
	SelectedLineBgColor             string            `json:"selectedLineBgColor"`
	InactiveViewSelectedLineBgColor string            `json:"inactiveViewSelectedLineBgColor"`
	VisualSelectionBgColor          string            `json:"visualSelectionBgColor"`
	PullRequestColors               []string          `json:"pullRequestColors"`
	ComponentColors                 map[string]string `json:"componentColors"`
}
