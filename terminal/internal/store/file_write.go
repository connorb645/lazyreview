package store

// FileWrite is one file a change writes: its path and its whole new text,
// or Remove to take the file away.
type FileWrite struct {
	Path   string
	Text   string
	Remove bool
}
