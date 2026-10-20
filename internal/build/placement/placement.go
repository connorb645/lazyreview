// Package placement puts each file of a change in a component and a
// package: by the config's path patterns, or by folder.
package placement

// Placement is the components that hold at least one file, each with its
// packages, and the place of each file, by its path.
type Placement struct {
	Components []Component
	Places     map[string]Place
}
