package config

// Section is a group of settings in their order: the lines of words before
// it, whether it is written on one line, and the note beside its name.
type Section struct {
	About   []string
	Flow    bool
	Note    string
	Entries []Entry
}
