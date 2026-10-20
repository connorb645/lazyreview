package languages

// Declaration is one type declaration a reader found: its name and kind,
// such as class or extension, a label such as "abstract class", the bare
// names it inherits from or conforms to, the byte offsets of its keyword,
// of its whole span and of its body's braces, how many braces enclose it,
// its first and last lines, whether it is a view or a test, the name of
// the declaration it sits in and of the outermost one, whether it is top
// level, and its stored properties.
type Declaration struct {
	Name       string     `json:"name"`
	Kind       string     `json:"kind"`
	Label      *string    `json:"label"`
	Inherits   []string   `json:"inherits"`
	At         int        `json:"at"`
	Span       [2]int     `json:"span"`
	Body       *[2]int    `json:"body"`
	Depth      int        `json:"depth"`
	Start      int        `json:"start"`
	End        int        `json:"end"`
	View       bool       `json:"view"`
	Test       bool       `json:"test"`
	Parent     *string    `json:"parent"`
	Owner      string     `json:"owner"`
	Top        bool       `json:"top"`
	Properties []Property `json:"properties"`
}
