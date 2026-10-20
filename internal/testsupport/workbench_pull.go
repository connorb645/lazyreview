package testsupport

// WorkbenchPull is a pull request of the workbench: its number, its base
// and head branches, its title, its author and its description.
type WorkbenchPull struct {
	Number int
	Base   string
	Head   string
	Title  string
	Author string
	Body   string
}
