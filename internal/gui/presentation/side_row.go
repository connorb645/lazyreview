package presentation

import "github.com/charmbracelet/lipgloss"

// SideRow is one row of a side panel: its key, its words, a title cut from
// its start when it must be, the words on its right, and what it stands
// for. A heading is not selectable; a lit row is in the highlight; an out
// row is one the filter leaves out.
type SideRow struct {
	Key        string
	Text       string
	Title      string
	TitleStyle lipgloss.Style
	Right      string
	Heading    bool
	Lit        bool
	Out        bool
	Item       Item
}
