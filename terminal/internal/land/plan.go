package land

// Plan is what landing a branch takes: the branch, the commit it is at and
// that commit's short id, the commits it brings to main here, newest
// first, as "<short id> <subject>", each remote with where main is on it,
// and the commit main is at here.
type Plan struct {
	Branch  string
	Commit  string
	Short   string
	Adds    []string
	Remotes []Remote
	Here    string
}
