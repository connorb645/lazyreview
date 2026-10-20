package testsupport

// FakeGit is what git said of the folder the stand-in Claude ran in: its
// branch, its branches with their commits, and its remotes.
type FakeGit struct {
	Branch   string `json:"branch"`
	Branches string `json:"branches"`
	Remotes  string `json:"remotes"`
}
