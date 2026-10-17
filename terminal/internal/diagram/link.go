package diagram

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// Link is a link from one card of a level to another, or to a part outside
// the level, which Outside names with its place, OutsideName and
// OutsidePlace being those two. AtRest links show without a focus: the
// structure and the links a config names. A link between components or
// packages counts the links between types in its Ways; Out says a pull
// request filter shows none of them.
type Link struct {
	From         string
	To           string
	Kind         string
	Words        string
	AtRest       bool
	Outside      string
	OutsideName  string
	OutsidePlace string
	Ways         []server.ArrowWay
	Out          bool
}
