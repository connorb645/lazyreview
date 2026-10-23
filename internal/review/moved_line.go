package review

import "github.com/connorb645/lazyreview/internal/models"

// MovedLine is where line of a file's old text sits in its new text, given
// the hunks of the diff between them: the line it became when the diff
// kept it; for a line the diff removed, the first line the diff puts after
// it in its place, else the last line before; and for a line outside every
// hunk, the same line moved by what the hunks above it added and removed.
func MovedLine(hunks []models.Hunk, line int) int {
	delta := 0
	for _, h := range hunks {
		oldEnd, newEnd := h.OldStart+h.OldCount, h.NewStart+h.NewCount
		if h.OldCount == 0 {
			oldEnd++
		}
		if h.NewCount == 0 {
			newEnd++
		}
		if line < oldEnd-h.OldCount {
			break
		}
		for i, r := range h.Rows {
			if r.Old != line {
				continue
			}
			if r.Kind == "same" {
				return r.New
			}
			return replacement(h, i)
		}
		delta = newEnd - oldEnd
	}
	return line + delta
}

func replacement(h models.Hunk, removed int) int {
	for _, r := range h.Rows[removed+1:] {
		if r.New > 0 {
			return r.New
		}
	}
	for i := removed - 1; i >= 0; i-- {
		if h.Rows[i].New > 0 {
			return h.Rows[i].New
		}
	}
	return max(1, h.NewStart)
}
