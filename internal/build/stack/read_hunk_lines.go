// Package stack ties each pull request of a stack to the stack's combined
// diff: where a comment on each line goes, and which pull requests changed
// each file.
package stack

import (
	"regexp"
	"strconv"
	"strings"
)

var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// ReadHunkLines reads a file's part of a diff, as diff.SplitDiff gives it,
// into its hunks with their line numbers.
func ReadHunkLines(text string) []HunkLines {
	var hunks []HunkLines
	var hunk *HunkLines
	oldLine, newLine := 0, 0
	for _, line := range strings.Split(text, "\n") {
		if m := hunkHeader.FindStringSubmatch(line); m != nil {
			hunks = append(hunks, HunkLines{
				OldStart: number(m[1]), OldLength: length(m[2]), NewStart: number(m[3]), NewLength: length(m[4]),
				Added: map[int]bool{}, Removed: map[int]bool{}, RemovedAt: map[int]int{}, NewToOld: map[int]int{}, OldToNew: map[int]int{},
			})
			hunk = &hunks[len(hunks)-1]
			oldLine, newLine = hunk.OldStart, hunk.NewStart
			continue
		}
		if hunk == nil || oldLine >= hunk.OldStart+hunk.OldLength && newLine >= hunk.NewStart+hunk.NewLength {
			continue
		}
		switch {
		case strings.HasPrefix(line, "+"):
			hunk.Added[newLine] = true
			newLine++
		case strings.HasPrefix(line, "-"):
			hunk.Removed[oldLine] = true
			hunk.RemovedAt[oldLine] = newLine
			oldLine++
		case strings.HasPrefix(line, " ") || line == "":
			hunk.NewToOld[newLine] = oldLine
			hunk.OldToNew[oldLine] = newLine
			oldLine++
			newLine++
		}
	}
	return hunks
}

// TowardOld says what happens to new-side line n going back to the old
// side.
func TowardOld(hunks []HunkLines, n int) LineStep {
	offset := 0
	for _, h := range hunks {
		start := rangeStart(h.NewStart, h.NewLength)
		if n < start {
			break
		}
		if n < start+h.NewLength {
			if h.Added[n] {
				return LineStep{Added: true}
			}
			return LineStep{Shown: true, Line: h.NewToOld[n]}
		}
		offset = rangeStart(h.OldStart, h.OldLength) + h.OldLength - (start + h.NewLength)
	}
	return LineStep{Line: n + offset}
}

// TowardNew says what happens to old-side line o going forward to the new
// side.
func TowardNew(hunks []HunkLines, o int) LineStep {
	offset := 0
	for _, h := range hunks {
		start := rangeStart(h.OldStart, h.OldLength)
		if o < start {
			break
		}
		if o < start+h.OldLength {
			if h.Removed[o] {
				return LineStep{Removed: true}
			}
			return LineStep{Shown: true, Line: h.OldToNew[o]}
		}
		offset = rangeStart(h.NewStart, h.NewLength) + h.NewLength - (start + h.OldLength)
	}
	return LineStep{Line: o + offset}
}

func rangeStart(start, length int) int {
	if length == 0 {
		return start + 1
	}
	return start
}

func number(text string) int {
	n, _ := strconv.Atoi(text)
	return n
}

func length(text string) int {
	if text == "" {
		return 1
	}
	return number(text)
}
