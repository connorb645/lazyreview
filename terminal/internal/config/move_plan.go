package config

// MovePlan is what the one-time move into the config file would do: the
// config file and whether it exists, settings.json and whether it exists,
// the values to write, holding only what differs from the defaults, where
// each came from, and every LAZYREVIEW_ variable set, written or not.
type MovePlan struct {
	ConfigPath     string
	ConfigExists   bool
	SettingsPath   string
	SettingsExists bool
	Values         YAMLMap
	From           []MoveSource
	Variables      []string
}
