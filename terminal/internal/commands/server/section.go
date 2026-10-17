package server

// Section is one ## section of a written review, with its file:line links.
type Section struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
	Links    []Link `json:"links"`
}
