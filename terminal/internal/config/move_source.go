package config

// MoveSource is where a value the move writes came from: a setting's
// dotted key, and the variable or the file that gave it.
type MoveSource struct {
	Key    string
	Source string
}
