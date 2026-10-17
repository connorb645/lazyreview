package types

// Edited is the editor closing on the file a comment, a question or an
// edit was written in.
type Edited struct {
	E    Editing
	File string
	Err  error
}
