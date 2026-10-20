package context

// The panels by number, as their titles and the keys 0 to 4 name them: the
// main panel on the right, and the four on the left.
const (
	MainPanel = iota
	ReviewPanel
	FilesPanel
	NotesPanel
	DiagramPanel
	SideCount = DiagramPanel
)
