// Package review reads what the terminal app shows of a review: its state,
// its page, each file ready to draw, its written review and the code a
// comment quotes, all from what the store keeps.
package review

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/models"
)

var hunkStart = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// ReadHunks is a file's part of a unified diff, with or without its header,
// hunk by hunk and row by row, as the app draws it: each row is add, del,
// same or note, a note being a line such as "\ No newline at end of file",
// with its line on each side, 0 on the side it is not on.
func ReadHunks(diff string) []models.Hunk {
	var hunks []models.Hunk
	at := -1
	old, now := 0, 0
	for _, line := range strings.Split(diff, "\n") {
		if m := hunkStart.FindStringSubmatch(line); m != nil {
			old, now = atoi(m[1]), atoi(m[3])
			hunks = append(hunks, models.Hunk{Header: line, OldStart: old, OldCount: count(m[2]), NewStart: now, NewCount: count(m[4]), Rows: []models.Row{}})
			at = len(hunks) - 1
			continue
		}
		if at < 0 {
			continue
		}
		h := &hunks[at]
		switch sign := firstByte(line); {
		case sign == '+':
			h.Rows = append(h.Rows, models.Row{Kind: "add", New: now, Text: line[1:]})
			now++
		case sign == '-':
			h.Rows = append(h.Rows, models.Row{Kind: "del", Old: old, Text: line[1:]})
			old++
		case sign == '\\':
			h.Rows = append(h.Rows, models.Row{Kind: "note", Text: line})
		case sign == ' ' || line == "" && !(now >= h.NewStart+h.NewCount && old >= h.OldStart+h.OldCount):
			h.Rows = append(h.Rows, models.Row{Kind: "same", Old: old, New: now, Text: rest(line)})
			old++
			now++
		}
	}
	return hunks
}

func firstByte(line string) byte {
	if line == "" {
		return 0
	}
	return line[0]
}

func rest(line string) string {
	if line == "" {
		return ""
	}
	return line[1:]
}

func count(text string) int {
	if text == "" {
		return 1
	}
	return atoi(text)
}

func atoi(text string) int {
	n, _ := strconv.Atoi(text)
	return n
}
