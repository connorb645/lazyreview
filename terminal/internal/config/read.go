package config

// Read is the config as read: the file's path and whether it exists, every
// value on top of the defaults, each section a map by name, where each
// setting's value came from, by its dotted key, and the settings the file
// gave that do nothing now.
type Read struct {
	Path    string
	Exists  bool
	Values  map[string]any
	Sources map[string]string
	Unused  []string
}
