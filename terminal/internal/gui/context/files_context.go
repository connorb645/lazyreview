package context

// FilesContext is the Files panel: its rows, and the components and folders
// folded in it.
type FilesContext struct {
	SideContext
	Folded map[string]bool
}
