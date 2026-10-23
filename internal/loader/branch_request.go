package loader

// BranchRequest is a load of a branch, or of a stack of branches: the
// folder of the repository, the base, and the heads, bottom first, each
// naming a branch, two or more being a stack, each based on the branch
// below; and the review's title. Config is a components config as JSON and
// Context whether to draw the unchanged code the change touches; each left
// nil keeps what the review had, or the default, and the title's default
// is the branch's name, or the stack's bottom and top.
type BranchRequest struct {
	Repo    string
	Base    string
	Heads   []string
	Title   *string
	Config  []byte
	Context *bool
}
