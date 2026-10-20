package review

import (
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/models"
)

// QuotedCode is the code a comment or a question on lines quotes, read from
// the review's page. Inside a hunk it gives the rows from the first line to
// the last: those of the side the lines are on, or every row between,
// removed and added, for a range from a removed line to an added one.
// Elsewhere, on the new side, it gives the lines of the file's whole text,
// when the page holds it, and otherwise nothing.
func QuotedCode(page *models.PageRecord, at models.Lines) string {
	startSide, first := at.Side, at.Line
	if at.StartLine != 0 {
		startSide, first = at.StartSide, at.StartLine
		if startSide == "" {
			startSide = at.Side
		}
	}
	keep := func(r models.Row) bool {
		other := "del"
		if at.Side == "LEFT" {
			other = "add"
		}
		return r.Kind != "note" && (startSide != at.Side || r.Kind != other)
	}
	for _, h := range ReadHunks(page.Diff[at.Path]) {
		from, to := rowOn(h.Rows, startSide, first), rowOn(h.Rows, at.Side, at.Line)
		if from >= 0 && to >= from {
			var text []string
			for _, r := range h.Rows[from : to+1] {
				if keep(r) {
					text = append(text, r.Text)
				}
			}
			return strings.Join(text, "\n")
		}
	}
	if startSide != "RIGHT" || at.Side != "RIGHT" {
		return ""
	}
	whole := WholeText(page, at.Path)
	if whole == nil {
		return ""
	}
	lo, hi := max(first-1, 0), min(at.Line, len(whole))
	if lo >= hi {
		return ""
	}
	return strings.Join(whole[lo:hi], "\n")
}

func rowOn(rows []models.Row, side string, line int) int {
	for i, r := range rows {
		if side == "LEFT" && r.Kind != "add" && r.Old == line || side != "LEFT" && r.Kind != "del" && r.New == line {
			return i
		}
	}
	return -1
}
