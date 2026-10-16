package app

// panelRow is one row of a panel: its words, a key or a name on the left,
// and what it opens, when it opens something.
type panelRow struct {
	left  string
	right string
	open  *item
}
