package land

// Remote is one remote of the checkout, by its name, with the commit main
// is at on it.
type Remote struct {
	Name string
	Main string
}
