package server

// Node is a type the diagram draws, or a file box for a file with no types.
// State is new, changed, same, removed or context; on a stack Pulls are the
// pull requests that changed it.
type Node struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Kind       string     `json:"kind"`
	Stereotype string     `json:"stereotype"`
	File       string     `json:"file"`
	Line       int        `json:"line"`
	Component  string     `json:"component"`
	Package    string     `json:"package"`
	State      string     `json:"state"`
	Group      string     `json:"group"`
	Pulls      []int      `json:"pulls"`
	Properties []Property `json:"properties"`
}
