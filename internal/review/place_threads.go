package review

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/connorb645/lazyreview/internal/models"
)

// PlaceThreads gives each thread At, the line the app shows it on, or nil
// when no line shown carries it: an outdated thread, one on the whole
// file, or on a stack one whose line a later pull request changed again. A
// single pull request's lines are the app's own; a stack's are found in its
// line map, from each pull request's own line back to the combined diff's.
func PlaceThreads(threads []models.GitHubThread, stack *models.StackRecord) []models.GitHubThread {
	var back map[string]models.Lines
	if stack != nil {
		back = ownToShown(stack.Lines)
	}
	out := make([]models.GitHubThread, len(threads))
	for i, t := range threads {
		t.At = placed(t, back, stack != nil)
		out[i] = t
	}
	return out
}

func placed(t models.GitHubThread, back map[string]models.Lines, stacked bool) *models.Lines {
	if t.Outdated || t.File || t.Line == 0 {
		return nil
	}
	startSide := t.StartSide
	if startSide == "" {
		startSide = t.Side
	}
	if !stacked {
		at := models.Lines{Path: t.Path, Side: t.Side, Line: t.Line}
		if t.StartLine != 0 {
			at.StartLine, at.StartSide = t.StartLine, startSide
		}
		return &at
	}
	end, found := back[threadKey(t.Pull, t.Side, t.Path, t.Line)]
	if !found {
		return nil
	}
	at := models.Lines{Path: end.Path, Side: t.Side, Line: end.Line}
	if t.StartLine != 0 {
		if start, found := back[threadKey(t.Pull, startSide, t.Path, t.StartLine)]; found && start.Path == end.Path {
			at.StartLine, at.StartSide = start.Line, startSide
		}
	}
	return &at
}

func ownToShown(lines map[string]map[string]models.LineMap) map[string]models.Lines {
	back := map[string]models.Lines{}
	add := func(place models.LinePlace, path, side string, line int) {
		own := place.Path
		if own == "" {
			own = path
		}
		k := threadKey(place.Pull, side, own, place.Line)
		if _, held := back[k]; !held {
			back[k] = models.Lines{Path: path, Line: line}
		}
	}
	paths := make([]string, 0, len(lines))
	for path := range lines {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		for _, side := range []string{"RIGHT", "LEFT"} {
			for _, line := range numbered(lines[path][side]) {
				entry := lines[path][side][strconv.Itoa(line)]
				switch {
				case entry.Owner != nil:
					add(*entry.Owner, path, side, line)
				case entry.Candidates != nil:
					for _, c := range entry.Candidates {
						add(c, path, side, line)
					}
				}
			}
		}
	}
	return back
}

func numbered(entries models.LineMap) []int {
	lines := make([]int, 0, len(entries))
	for key := range entries {
		if n, err := strconv.Atoi(key); err == nil {
			lines = append(lines, n)
		}
	}
	sort.Ints(lines)
	return lines
}

func threadKey(pull int, side, path string, line int) string {
	return fmt.Sprintf("%d %s %d %s", pull, side, line, path)
}
