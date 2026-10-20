package languages

// Family is the words that declare a type in a family of languages, and
// the globs of the files that hold its code.
type Family struct {
	Keywords []string
	Globs    []string
}
