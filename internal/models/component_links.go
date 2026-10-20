package models

// ComponentLinks is every link from the types of one component to those of
// another, with its count and the words a config gives it, which the arrow
// between the two components carries.
type ComponentLinks struct {
	Source string      `json:"source"`
	Target string      `json:"target"`
	Count  int         `json:"count"`
	Label  string      `json:"label"`
	Links  [][2]string `json:"links"`
}
