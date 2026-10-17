// Package diagram arranges the diagram for a terminal in its three levels:
// the components, the packages of one component, or the types of one
// package, as rows and columns of cards read from the layout's own places.
package diagram

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// Card is one box of a level: a component, a package, a type, a file with
// no types, or a group of look-alike types. Title and Detail are its two
// lines; State is a type's new, changed, same, removed or context. Pulls
// are the pull requests that changed it, or anything it holds.
type Card struct {
	ID      string
	Kind    string
	Title   string
	Detail  string
	State   string
	Pulls   []int
	Context bool
	Area    string
	Members []string
	Box     server.Box
}
