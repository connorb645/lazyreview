package git

// CopyOf is what a copy of the code holds: the bare clone it borrows from,
// the commit it is at, a branch for each name, at its commit, and the
// branch checked out, one of them, at the commit.
type CopyOf struct {
	Source   string
	Commit   string
	Branches map[string]string
	Branch   string
}
