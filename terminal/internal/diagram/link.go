package diagram

// Link is a link from one card of a level to another, or to a part outside
// the level, which Outside names with its place, OutsideName and
// OutsidePlace being those two. AtRest links show without a focus: the
// structure and the links a config names.
type Link struct {
	From         string
	To           string
	Kind         string
	Words        string
	AtRest       bool
	Outside      string
	OutsideName  string
	OutsidePlace string
}
