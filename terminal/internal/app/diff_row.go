package app

import "review-desk/terminal/internal/server"

// diffRow is one row of the diff view: a file's header, a hunk's header, a
// folded run of unchanged lines, a note about a file, a line, which the
// split view draws as a pair, its removed side on the left, a drawn line of
// a block of a Markdown file's Rendered view, or the head or a line of text
// of a note on a line. Key names the hunk a header heads,
// the run of unchanged lines a row is in, or the note a row is part of.
type diffRow struct {
	kind    string
	path    string
	text    string
	key     string
	count   int
	line    *server.Row
	left    *server.Row
	right   *server.Row
	added   bool
	part    string
	message int
	block   *server.Block
	last    bool
}

// at is the side and line a comment on the row goes on, as the app shows
// it, or false for a row that is no line.
func (r diffRow) at() (string, int, bool) {
	switch {
	case r.line != nil && r.line.Kind == "del":
		return "LEFT", r.line.Old, true
	case r.line != nil && r.line.Kind != "note":
		return "RIGHT", r.line.New, true
	case r.right != nil:
		return "RIGHT", r.right.New, true
	case r.left != nil:
		return "LEFT", r.left.Old, true
	}
	return "", 0, false
}

// words is what the server says about the row's line: who changed it and
// where a comment on it goes, on a stack.
func (r diffRow) words() string {
	for _, l := range []*server.Row{r.line, r.right, r.left} {
		if l != nil && l.Words != "" {
			return l.Words
		}
	}
	return ""
}
