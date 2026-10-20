package placement

// Grouping says how GroupByFolder splits: into at most MaxGroups groups, a
// group splitting only when it holds more than SplitAbove paths and, when
// Balanced, only when at least two of its subfolders hold a real share of
// it. With SplitTop the top folder always splits into its subfolders.
type Grouping struct {
	MaxGroups  int
	SplitAbove float64
	Balanced   bool
	SplitTop   bool
}
