package languages

// Property is a stored property of a type: its name, its written type, or
// the type its initializer constructs with Inferred true, or nil, and the
// line it is declared on.
type Property struct {
	Name     string  `json:"name"`
	Type     *string `json:"type"`
	Line     int     `json:"line"`
	Inferred bool    `json:"inferred"`
}
