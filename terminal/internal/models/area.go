package models

// Area frames some components of the diagram, as a config names them, with
// the components it holds; each component names its area.
type Area struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Components  []string `json:"components"`
}
