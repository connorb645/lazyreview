package components

// ContextTypes says which unchanged types to draw around the change: the
// names always drawn, the names never drawn, and at most how many.
type ContextTypes struct {
	Include []string
	Exclude []string
	Max     int
}
