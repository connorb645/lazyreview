package types

// DiagramPlace is a box of the diagram: its level, the component and the
// package the level is in, and the box's id.
type DiagramPlace struct {
	Level     string
	Component string
	Package   string
	Box       string
}
