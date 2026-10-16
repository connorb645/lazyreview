package app

// route is a link as drawn between two cards of a level: its words, whether
// the focus lights it or only every link shows it, and how it runs. Cards
// side by side join straight across, on line `across` of their row. Other
// links leave a border at column xa and reach one at column xb, running
// down, across and down in a lane of each gap they cross, and past the rows
// between at column via. A lane of -1 is a straight run with no words.
type route struct {
	from, to int
	words    string
	lit      bool
	faint    bool
	side     bool
	across   int
	gaps     []int
	lanes    []int
	via      int
	xa, xb   int
}
