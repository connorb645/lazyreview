package app

// diagramPlace is a box of the diagram: the level it sits on, the component
// and package that level shows, and the box's id.
type diagramPlace struct {
	level     string
	component string
	pkg       string
	box       string
}
