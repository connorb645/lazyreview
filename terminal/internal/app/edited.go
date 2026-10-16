package app

// edited says the editor has closed on a file.
type edited struct {
	e    editing
	file string
	err  error
}
