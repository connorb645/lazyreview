package loader

// BranchRequest is a load of a branch: the folder of the repository, the
// base and head, which must name a branch, and the review's title. Config
// is a components config as JSON and Context whether to draw the unchanged
// code the change touches; each left nil keeps what the review had, or the
// default, and the title's default is the branch's name.
type BranchRequest struct {
	Repo    string
	Base    string
	Head    string
	Title   *string
	Config  []byte
	Context *bool
}
