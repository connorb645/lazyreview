package placement

// Component is a component that holds at least one of the files placed:
// as the config declares it, or made from a folder of files the config
// does not name. Size is how many of the files it holds.
type Component struct {
	ID          string
	Title       string
	Description string
	Tag         string
	Color       string
	Group       string
	Context     bool
	Declared    bool
	Size        int
	Packages    []Package
}
