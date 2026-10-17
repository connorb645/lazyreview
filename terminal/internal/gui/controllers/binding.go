// Package controllers gives each context its bindings: the actions it
// takes, the keys bound to them and what each does, with the keys popup's
// help and the bar's hints for it, made from those bindings.
package controllers

// Binding is one action of a context: its name, the key sequences bound to
// it as keys.Reader names them, such as "g g" or "ctrl+o", the first being
// the one the help and the bar show, what it does, and its words in the
// list after Space.
type Binding struct {
	Action      string
	Keys        []string
	Handler     Handler
	Description string
}
