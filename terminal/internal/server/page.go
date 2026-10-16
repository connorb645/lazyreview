package server

// Page is what a review draws: the build id that every change made from it
// names, and the model of the change.
type Page struct {
	Build string     `json:"build"`
	Model Model      `json:"model"`
	Stack *PageStack `json:"stack"`
}
