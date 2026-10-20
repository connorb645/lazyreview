package components

// Link is a link between two types the config names, with its words; Kind
// is use or has.
type Link struct {
	From  string
	To    string
	Label string
	Kind  string
}
