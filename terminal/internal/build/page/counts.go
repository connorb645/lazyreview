package page

// Counts is what a page drew: its files, their added and removed lines,
// its components and its types.
type Counts struct {
	Files      int `json:"files"`
	Adds       int `json:"adds"`
	Dels       int `json:"dels"`
	Components int `json:"components"`
	Types      int `json:"types"`
}
