package models

// Layout is where the diagram places each part, in the layout's
// own units: each component, each package frame, type and group by id, and
// the arrows between packages and between components.
type Layout struct {
	Components map[string]Box   `json:"components"`
	Boxes      map[string]Box   `json:"boxes"`
	PkgArrows  []PackageArrow   `json:"pkgArrows"`
	Summaries  []ComponentArrow `json:"summaries"`
}
