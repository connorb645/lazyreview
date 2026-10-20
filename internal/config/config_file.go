package config

// ConfigFile is where the config file is, and whether it exists there.
type ConfigFile struct {
	Path   string
	Exists bool
}
