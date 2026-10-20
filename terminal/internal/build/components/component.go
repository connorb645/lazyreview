package components

// Component is a part of the change the config names, with the path
// patterns it holds, directly or through its packages. A context component
// holds only unchanged code drawn for context.
type Component struct {
	ID          string
	Title       string
	Description string
	Tag         string
	Color       string
	Group       string
	Context     bool
	Paths       []string
	Packages    []Package
}
