package server

// Property is a stored property of a type: its name, its type, written or
// inferred from what its initializer constructs, and its line.
type Property struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Line     int    `json:"line"`
	Inferred bool   `json:"inferred"`
}
