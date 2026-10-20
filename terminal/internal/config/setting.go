package config

// Setting is one setting of the config: how its value is checked, its
// default, the variable that overrides it, the lines of words that say what
// it does and of an example after it, and the words a choice takes.
type Setting struct {
	Check   string
	Default any
	Env     string
	About   []string
	Example []string
	Choices []string
}
