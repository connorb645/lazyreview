package config

// Leaf is one setting with its dotted key, such as "claude.command".
type Leaf struct {
	Key     string
	Setting *Setting
}
