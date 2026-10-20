package config

// Entry is one setting or group of settings of a section, by its name.
type Entry struct {
	Name    string
	Setting *Setting
	Section *Section
}
