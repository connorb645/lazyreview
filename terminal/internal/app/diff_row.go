package app

import "github.com/connorb645/lazyreview/terminal/internal/server"

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

func (r diffRow) words() string {
	for _, l := range []*server.Row{r.line, r.right, r.left} {
		if l != nil && l.Words != "" {
			return l.Words
		}
	}
	return ""
}
