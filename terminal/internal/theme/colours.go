package theme

// Colours are the terminal app's colours, one for each role, each written
// as #rrggbb or as a terminal colour number. The names are lazygit's where
// the role is the same.
type Colours struct {
	DefaultFgColor                  string
	MutedFgColor                    string
	FaintFgColor                    string
	AccentColor                     string
	ActiveBorderColor               string
	InactiveBorderColor             string
	AddedColor                      string
	RemovedColor                    string
	WarningColor                    string
	SelectedLineBgColor             string
	InactiveViewSelectedLineBgColor string
	VisualSelectionBgColor          string
	PullRequestColors               []string
	ComponentColors                 map[string]string
}
