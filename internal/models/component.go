package models

// Component is a part of the change the diagram frames, such as an app's
// module, with the folders it holds. A context component holds unchanged
// code the change touches.
type Component struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Tag         string    `json:"tag"`
	Color       string    `json:"color"`
	Area        string    `json:"area"`
	Context     bool      `json:"context"`
	Packages    []Package `json:"packages"`
}
