package languages

// Reading is what a reader finds in a file's text: the text with its
// comments and the insides of its strings blanked, at the same byte
// offsets, every type declaration in source order, and the imports.
type Reading struct {
	Masked       string
	Declarations []Declaration
	Imports      []Import
}
