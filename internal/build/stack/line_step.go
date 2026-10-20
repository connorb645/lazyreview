package stack

// LineStep is what happens to a line going from one side of a diff to the
// other: Added or Removed when the diff adds or removes it, and otherwise
// its number on the other side, with Shown when a hunk shows it unchanged.
type LineStep struct {
	Added   bool
	Removed bool
	Shown   bool
	Line    int
}
