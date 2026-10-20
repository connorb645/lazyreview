package languages

// Language is a language the map reads types from: its name, its family,
// within which types resolve across files, the file extensions it is
// chosen by, and its reader.
type Language struct {
	Name       string
	Family     string
	Extensions []string
	Read       func(text string) Reading
}
