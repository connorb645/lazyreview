package config

// Leaf is one setting with its dotted key, such as "server.port".
type Leaf struct {
	Key     string
	Setting *Setting
}
