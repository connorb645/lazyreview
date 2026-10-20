package git

// GrepLine is one line a search found: the file's path and the line.
type GrepLine struct {
	Path string
	Text string
}
