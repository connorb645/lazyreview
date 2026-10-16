package server

// Edge is a link between two types: realize, inherit, has, extend, use or
// test. A has link names the properties that hold its target; Key marks a
// link the config names, with its words in Label.
type Edge struct {
	Source     string   `json:"source"`
	Target     string   `json:"target"`
	Kind       string   `json:"kind"`
	Label      string   `json:"label"`
	Properties []string `json:"properties"`
	Key        bool     `json:"key"`
}
