package config

// KeybindingReviewScreenConfig is the keys every place of a review takes.
type KeybindingReviewScreenConfig struct {
	NextPull     []string `json:"nextPull"`
	PreviousPull []string `json:"previousPull"`
	JumpBack     []string `json:"jumpBack"`
	JumpForward  []string `json:"jumpForward"`
	Diagram      []string `json:"diagram"`
	Finder       []string `json:"finder"`
	Home         []string `json:"home"`
	FocusMain    []string `json:"focusMain"`
}
