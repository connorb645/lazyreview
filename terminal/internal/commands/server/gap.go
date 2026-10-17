package server

// Gap is a run of unchanged lines between hunks: it comes before hunk
// Before, runs from new line From to To, and each of its lines is old line
// Delta further on.
type Gap struct {
	Before int `json:"before"`
	From   int `json:"from"`
	To     int `json:"to"`
	Delta  int `json:"delta"`
}
