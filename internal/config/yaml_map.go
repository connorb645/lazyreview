package config

// YAMLMap is a YAML mapping as written: its keys in the order they come,
// and the value of each.
type YAMLMap struct {
	Keys   []string
	Values map[string]any
}
