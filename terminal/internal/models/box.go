package models

// Box is where a part sits on the diagram's canvas: its left edge, its top
// edge and its height, which set its row and its order in the row.
type Box struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	H float64 `json:"h"`
}
