package presentation

// LineNumbers is how the diff numbers its lines, as the config's
// gui.lineNumbers says: each line its own number, or the cursor's line its
// own and every other line its distance from it, as Vim's relativenumber
// does with number, or no numbers at all.
type LineNumbers string

// The ways the diff numbers its lines.
const (
	AbsoluteNumbers LineNumbers = "absolute"
	RelativeNumbers LineNumbers = "relative"
	NoNumbers       LineNumbers = "none"
)
