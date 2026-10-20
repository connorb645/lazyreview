package review

import (
	"strconv"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
)

// StackPlace is where a stack's line map sends a comment on one line of the
// combined diff: the pull request that last changed it, at its own path,
// line and side; the pull requests whose own diffs show it unchanged; or
// why none can take one.
type StackPlace struct {
	Pull       int
	Path       string
	Line       int
	Side       string
	Candidates []StackPlace
	None       string
}

// LineEntry reads one entry of a stack's line map back, for line of path
// on side as the combined diff numbers it.
func LineEntry(tr *i18n.TranslationSet, lines map[string]map[string]models.LineMap, path string, line int, side string) StackPlace {
	entry, ok := lines[path][side][strconv.Itoa(line)]
	switch {
	case !ok:
		return StackPlace{None: tr.NotInCombinedDiff}
	case entry.Owner != nil:
		return placeOf(*entry.Owner, path, side)
	case entry.Candidates != nil:
		candidates := make([]StackPlace, len(entry.Candidates))
		for i, c := range entry.Candidates {
			candidates[i] = placeOf(c, path, side)
		}
		return StackPlace{Candidates: candidates}
	}
	return StackPlace{None: entry.None}
}

func placeOf(p models.LinePlace, path, side string) StackPlace {
	if p.Path != "" {
		path = p.Path
	}
	return StackPlace{Pull: p.Pull, Path: path, Line: p.Line, Side: side}
}
