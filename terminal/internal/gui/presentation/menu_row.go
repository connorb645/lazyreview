package presentation

// MenuRow is one row of a menu: its key or kind on the left, its words on
// the right, or a heading, and what Enter on it opens.
type MenuRow struct {
	Left    string
	Right   string
	Heading string
	Open    *Item
}
