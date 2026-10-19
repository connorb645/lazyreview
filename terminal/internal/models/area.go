package models

// Area frames some components of the diagram, as a config names them; each
// component names its area.
type Area struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
