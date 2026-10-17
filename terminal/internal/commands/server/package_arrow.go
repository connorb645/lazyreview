package server

// PackageArrow is the one arrow between two packages of a component, with
// its words and its count of links.
type PackageArrow struct {
	Component string `json:"comp"`
	From      string `json:"from"`
	To        string `json:"to"`
	Words     string `json:"words"`
	CountText string `json:"countText"`
}
