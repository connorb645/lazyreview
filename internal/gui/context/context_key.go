package context

// ContextKey names a context: one for each view, and the global one.
type ContextKey string

// The contexts of the gui.
const (
	HomeKey                ContextKey = "home"
	HomeMainKey            ContextKey = "homeMain"
	ReviewPanelKey         ContextKey = "reviewPanel"
	FilesPanelKey          ContextKey = "filesPanel"
	NotesPanelKey          ContextKey = "notesPanel"
	DiagramPanelKey        ContextKey = "diagramPanel"
	ReviewViewKey          ContextKey = "reviewView"
	DiffKey                ContextKey = "diff"
	DiagramKey             ContextKey = "diagram"
	MenuKey                ContextKey = "menu"
	FinderKey              ContextKey = "finder"
	LabelsKey              ContextKey = "labels"
	PostingKey             ContextKey = "posting"
	PostingConfirmationKey ContextKey = "postingConfirmation"
	ConfirmationKey        ContextKey = "confirmation"
	PromptKey              ContextKey = "prompt"
	GlobalKey              ContextKey = "global"
)

// Kind is what kind of context a key names.
func (k ContextKey) Kind() Kind {
	switch k {
	case HomeKey, ReviewPanelKey, FilesPanelKey, NotesPanelKey, DiagramPanelKey:
		return SideKind
	case HomeMainKey, ReviewViewKey, DiffKey, DiagramKey:
		return MainKind
	case GlobalKey:
		return GlobalKind
	}
	return PopupKind
}
