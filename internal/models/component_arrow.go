package models

// ComponentArrow is the one arrow between two components, both ways when
// Both, with the lines of words it carries for the whole stack and each
// way's links.
type ComponentArrow struct {
	Source string     `json:"source"`
	Target string     `json:"target"`
	Both   bool       `json:"both"`
	Lines  []string   `json:"lines"`
	Ways   []ArrowWay `json:"ways"`
}
