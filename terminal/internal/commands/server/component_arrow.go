package server

// ComponentArrow is the one arrow between two components, both ways when
// Both, with the lines of words it carries.
type ComponentArrow struct {
	Source string   `json:"source"`
	Target string   `json:"target"`
	Both   bool     `json:"both"`
	Lines  []string `json:"lines"`
}
