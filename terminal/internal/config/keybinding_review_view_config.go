package config

// KeybindingReviewViewConfig is the keys of the Review view in the main
// panel.
type KeybindingReviewViewConfig struct {
	Down            []string `json:"down"`
	Up              []string `json:"up"`
	Top             []string `json:"top"`
	Bottom          []string `json:"bottom"`
	HalfDown        []string `json:"halfDown"`
	HalfUp          []string `json:"halfUp"`
	PageDown        []string `json:"pageDown"`
	PageUp          []string `json:"pageUp"`
	NextSection     []string `json:"nextSection"`
	PreviousSection []string `json:"previousSection"`
	Fold            []string `json:"fold"`
	Open            []string `json:"open"`
	Edit            []string `json:"edit"`
	Drop            []string `json:"drop"`
	Post            []string `json:"post"`
}
