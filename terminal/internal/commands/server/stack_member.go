package server

// StackMember is one pull request of a stack, with the size of its own diff.
type StackMember struct {
	Pull  int    `json:"pull"`
	Head  string `json:"head"`
	Files int    `json:"files"`
	Adds  int    `json:"adds"`
	Dels  int    `json:"dels"`
}
