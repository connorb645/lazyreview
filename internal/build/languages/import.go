package languages

// Import is one import of a file: the module it names, the names it takes
// from it, or nil for every name, and whether it is Python's.
type Import struct {
	Spec   string   `json:"spec"`
	Names  []string `json:"names"`
	Python bool     `json:"python,omitempty"`
}
