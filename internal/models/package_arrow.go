package models

// PackageArrow is the one arrow between two packages of a component, with
// its words, its count of links and the links between types it counts.
type PackageArrow struct {
	Component string      `json:"comp"`
	From      string      `json:"from"`
	To        string      `json:"to"`
	Words     string      `json:"words"`
	CountText string      `json:"countText"`
	Links     [][2]string `json:"links"`
}
