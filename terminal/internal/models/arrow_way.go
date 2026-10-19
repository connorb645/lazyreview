package models

// ArrowWay is one way of an arrow between components: its arrow, its label
// and the links between types it counts, as source and target.
type ArrowWay struct {
	Arrow string      `json:"arrow"`
	Label string      `json:"label"`
	Links [][2]string `json:"links"`
}
