package models

// Page is what a review draws: the build id that every change made from it
// names, the model of the change and where the diagram places its parts.
type Page struct {
	Build  string     `json:"build"`
	Model  Model      `json:"model"`
	Layout Layout     `json:"layout"`
	Stack  *PageStack `json:"stack"`
}
