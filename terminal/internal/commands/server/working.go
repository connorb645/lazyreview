package server

// Working is what Claude has done so far on a conversation it is answering.
type Working struct {
	Steps []Step `json:"steps"`
}
