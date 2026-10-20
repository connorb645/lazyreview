package model

// ChangedTypes is the types the changed files declare: Nodes, one per type
// the change adds or edits, one per file of extensions to types it does not
// draw, and one box per file it draws nothing else for; Untouched, the
// other top-level types in those files, which may still be drawn as
// unchanged context; and each file's imports.
type ChangedTypes struct {
	Nodes     []*Node
	Untouched []*Node
	Imports   []FileImports
}
