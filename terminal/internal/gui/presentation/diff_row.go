package presentation

import "github.com/connorb645/lazyreview/terminal/internal/models"

// DiffRow is one row of a diff: a file's header, a hunk's, a run of
// unchanged lines, a line, a pair of lines side by side, a block of a
// rendered file, a note's head or text, or a remark.
type DiffRow struct {
	Kind    string
	Path    string
	Text    string
	Key     string
	Count   int
	Line    *models.Row
	Left    *models.Row
	Right   *models.Row
	Added   bool
	Part    string
	Message int
	Block   *models.Block
	Last    bool
}

// At is the side and number of the line the row shows, if it shows one.
func (r DiffRow) At() (string, int, bool) {
	switch {
	case r.Line != nil && r.Line.Kind == "del":
		return "LEFT", r.Line.Old, true
	case r.Line != nil && r.Line.Kind != "note":
		return "RIGHT", r.Line.New, true
	case r.Right != nil:
		return "RIGHT", r.Right.New, true
	case r.Left != nil:
		return "LEFT", r.Left.Old, true
	}
	return "", 0, false
}

// Words are what the queue says of the row's line, if anything.
func (r DiffRow) Words() string {
	for _, l := range []*models.Row{r.Line, r.Right, r.Left} {
		if l != nil && l.Words != "" {
			return l.Words
		}
	}
	return ""
}
