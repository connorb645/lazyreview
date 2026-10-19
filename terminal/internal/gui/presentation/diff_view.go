package presentation

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// DiffView is a diff as it is drawn: the files loaded, which of them and of
// their runs and notes are folded, open, whole or rendered, split or wrapped
// in a width, how its lines are numbered, its rows, the cursor and the first
// row shown, a selection from Anchor to the cursor, and the columns of code
// scrolled out of sight on the left.
type DiffView struct {
	Files     map[string]server.File
	Folded    map[string]bool
	Opened    map[string]bool
	Whole     map[string]bool
	Rendered  map[string]bool
	Notes     map[string]bool
	Split     bool
	Wrap      bool
	Width     int
	Numbers   LineNumbers
	Rows      []DiffRow
	Cursor    int
	Top       int
	Selecting bool
	Anchor    int
	Left      int
}
