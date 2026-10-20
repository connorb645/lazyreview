package components

// Package is a folder of a component the config names, by path patterns.
type Package struct {
	ID          string
	Title       string
	Description string
	Paths       []string
}
