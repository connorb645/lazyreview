package models

// Set is one action on offer for a pull request: the id of exactly the
// comments a press sends, and the sentence its press shows before it posts.
type Set struct {
	Set   string `json:"set"`
	Words string `json:"words"`
}
