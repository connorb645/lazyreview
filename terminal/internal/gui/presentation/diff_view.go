package presentation

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// DiffView is a component's diff as it is drawn: its files, which of them
// and of their runs and notes are folded, open, whole or rendered, split or
// wrapped in a width, how its lines are numbered, its rows, the cursor and
// the first row shown, and a selection from Anchor to the cursor.
type DiffView struct {
	Component string
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
}
