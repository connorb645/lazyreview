// Package diff reads git's diff of a change: one chunk per file, and the
// text of each file it touches.
package diff

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	hunkHeader   = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
	quotedHeader = regexp.MustCompile(`^("(?:[^"\\]|\\.)*") ("(?:[^"\\]|\\.)*"|.*)$`)
	plainHeader  = regexp.MustCompile(`^a/(.*) b/(.*)$`)
)

// SplitDiff splits the text of git diff into one chunk per file.
func SplitDiff(text string) []Chunk {
	lines := strings.Split(text, "\n")
	var chunks []Chunk
	start := -1
	for i := 0; i <= len(lines); i++ {
		if i == len(lines) || strings.HasPrefix(lines[i], "diff --git ") {
			if start >= 0 {
				chunks = append(chunks, readChunk(lines[start:trimEnd(lines, start, i)]))
			}
			start = i
		}
	}
	return chunks
}

// Touches says whether hunks add a line from start to end, counted from 1
// on the new side and both included, or remove lines from between two of
// them.
func Touches(hunks []Hunk, start, end int) bool {
	for _, h := range hunks {
		for _, n := range h.Added {
			if n >= start && n <= end {
				return true
			}
		}
		for _, n := range h.RemovedAt {
			if n > start && n <= end {
				return true
			}
		}
	}
	return false
}

func trimEnd(lines []string, start, end int) int {
	for end > start+1 && lines[end-1] == "" {
		end--
	}
	return end
}

func unquote(path string) string {
	if !strings.HasPrefix(path, `"`) || len(path) < 2 {
		return path
	}
	inner := path[1 : len(path)-1]
	var out []byte
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if c != '\\' || i+1 == len(inner) {
			out = append(out, c)
			continue
		}
		if i+3 < len(inner) && isOctal(inner[i+1]) && isOctal(inner[i+2]) && isOctal(inner[i+3]) {
			b, _ := strconv.ParseUint(inner[i+1:i+4], 8, 8)
			out = append(out, byte(b))
			i += 3
			continue
		}
		switch inner[i+1] {
		case 'n':
			out = append(out, '\n')
		case 't':
			out = append(out, '\t')
		default:
			out = append(out, inner[i+1])
		}
		i++
	}
	return string(out)
}

func isOctal(c byte) bool {
	return c >= '0' && c <= '7'
}

func stripPrefix(path, prefix string) string {
	if tab := strings.IndexByte(path, '\t'); tab >= 0 {
		path = path[:tab]
	}
	return strings.TrimPrefix(unquote(path), prefix)
}

func headerPaths(header string) (string, string) {
	rest := strings.TrimPrefix(header, "diff --git ")
	if strings.HasPrefix(rest, `"`) {
		if m := quotedHeader.FindStringSubmatch(rest); m != nil {
			return stripPrefix(m[1], "a/"), stripPrefix(m[2], "b/")
		}
	}
	if half := (len(rest) - 1) / 2; (len(rest)-1)%2 == 0 && half >= 2 && rest[2:half] == rest[half+3:] {
		return rest[2:half], rest[half+3:]
	}
	if m := plainHeader.FindStringSubmatch(rest); m != nil {
		return m[1], m[2]
	}
	return rest, rest
}

func readChunk(lines []string) Chunk {
	headerOld, headerNew := headerPaths(lines[0])
	var oldPath, newPath string
	status := "M"
	binary := false
	var hunks []Hunk
	var hunk *Hunk
	newLine := 0
	for _, line := range lines[1:] {
		if hunk == nil {
			switch {
			case strings.HasPrefix(line, "new file mode"):
				status = "A"
			case strings.HasPrefix(line, "deleted file mode"):
				status = "D"
			case strings.HasPrefix(line, "rename from "):
				status, oldPath = "R", unquote(line[len("rename from "):])
			case strings.HasPrefix(line, "rename to "):
				newPath = unquote(line[len("rename to "):])
			case strings.HasPrefix(line, "copy from "):
				status, oldPath = "A", unquote(line[len("copy from "):])
			case strings.HasPrefix(line, "copy to "):
				newPath = unquote(line[len("copy to "):])
			case strings.HasPrefix(line, "Binary files ") || line == "GIT binary patch":
				binary = true
			case strings.HasPrefix(line, "--- "):
				if p := line[4:]; p != "/dev/null" {
					oldPath = stripPrefix(p, "a/")
				}
			case strings.HasPrefix(line, "+++ "):
				if p := line[4:]; p != "/dev/null" {
					newPath = stripPrefix(p, "b/")
				}
			}
		}
		if m := hunkHeader.FindStringSubmatch(line); m != nil {
			hunks = append(hunks, Hunk{OldStart: number(m[1]), OldLength: length(m[2]), NewStart: number(m[3]), NewLength: length(m[4])})
			hunk = &hunks[len(hunks)-1]
			newLine = hunk.NewStart
			continue
		}
		if hunk == nil {
			continue
		}
		switch {
		case strings.HasPrefix(line, "+"):
			hunk.Added = append(hunk.Added, newLine)
			newLine++
		case strings.HasPrefix(line, "-"):
			hunk.RemovedAt = append(hunk.RemovedAt, newLine)
		case strings.HasPrefix(line, " ") || line == "":
			newLine++
		}
	}
	if status == "D" {
		newPath = ""
	}
	path := newPath
	if path == "" {
		path = oldPath
	}
	if path == "" {
		path = headerNew
	}
	before := path
	switch status {
	case "R", "D":
		before = oldPath
		if before == "" {
			before = headerOld
		}
	case "A":
		before = ""
	}
	return Chunk{Path: path, OldPath: before, Status: status, Binary: binary, Text: strings.Join(lines, "\n"), Hunks: hunks}
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
