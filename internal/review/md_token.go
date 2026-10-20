package review

// mdToken is one top-level block of a text: its source with no trailing
// space, the lines it spans, and for a list whether it is ordered, the
// number it starts from and each item's source and lines.
type mdToken struct {
	raw     string
	span    mdSpan
	list    bool
	ordered bool
	start   int
	items   []mdToken
}
