package context

// Kind is a context's kind, as lazygit has them: a panel on the left, the
// main panel, a popup over them, or the global context, which has no view.
type Kind int

// The kinds of context.
const (
	SideKind Kind = iota
	MainKind
	PopupKind
	GlobalKind
)
