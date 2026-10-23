package stack

import (
	"strconv"

	"github.com/connorb645/lazyreview/internal/build/diff"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
)

// MapLines traces every line the hunks of chunks, the stack's combined diff,
// show, through steps, bottom first, to the member of pulls that takes a
// comment on it. A file with no lines to trace, such as a binary file or a
// pure rename, is owned by every member whose own diff touches it.
func MapLines(tr *i18n.TranslationSet, steps []Step, chunks []diff.Chunk, pulls []int) Mapped {
	mapped := Mapped{Lines: map[string]map[string]models.LineMap{}, Files: map[string]FilePulls{}}
	notAdded, notRemoved := tr.StackNotAdded, tr.StackNotRemoved
	if len(steps) > 0 && steps[0].Branch != "" {
		notAdded, notRemoved = tr.StackBranchNotAdded, tr.StackBranchNotRemoved
	}
	for _, chunk := range chunks {
		right, left := models.LineMap{}, models.LineMap{}
		byPull := map[int][2]int{}
		count := func(pull, k int) {
			c := byPull[pull]
			c[k]++
			byPull[pull] = c
		}
		for _, h := range ReadHunkLines(chunk.Text) {
			for n := range h.Added {
				found := traceNew(steps, chunk.Path, n)
				if found.owner == nil {
					right[itoa(n)] = models.LineEntry{None: notAdded}
					continue
				}
				right[itoa(n)] = owned(*found.owner, chunk.Path)
				count(found.owner.pull, 0)
			}
			for o := range h.Removed {
				found := traceOld(steps, chunk.OldPath, o)
				if found.owner == nil {
					left[itoa(o)] = models.LineEntry{None: notRemoved}
					continue
				}
				left[itoa(o)] = owned(*found.owner, chunk.Path)
				count(found.owner.pull, 1)
			}
			for n, o := range h.NewToOld {
				right[itoa(n)] = unchanged(traceNew(steps, chunk.Path, n), chunk.Path)
				left[itoa(o)] = unchanged(traceOld(steps, chunk.OldPath, o), chunk.Path)
			}
		}
		mapped.Lines[chunk.Path] = map[string]models.LineMap{"RIGHT": right, "LEFT": left}
		owners := []int{}
		if len(byPull) > 0 {
			for _, p := range pulls {
				if _, ok := byPull[p]; ok {
					owners = append(owners, p)
				}
			}
		} else {
			owners = touching(steps, chunk)
		}
		mapped.Files[chunk.Path] = FilePulls{Pulls: owners, ByPull: byPull}
	}
	return mapped
}

// ChangedInSpan is the pull requests that changed a line from start to end
// of a file, counted from 1 on the new side and both included, or removed
// one from between two of them, as the file's hunks and line entries in the
// combined diff show it.
func ChangedInSpan(hunks []HunkLines, lines map[string]models.LineMap, start, end int) map[int]bool {
	found := map[int]bool{}
	for _, h := range hunks {
		for n := range h.Added {
			if entry, ok := lines["RIGHT"][itoa(n)]; ok && n >= start && n <= end && entry.Owner != nil {
				found[entry.Owner.Pull] = true
			}
		}
		for o, at := range h.RemovedAt {
			if entry, ok := lines["LEFT"][itoa(o)]; ok && at > start && at <= end && entry.Owner != nil {
				found[entry.Owner.Pull] = true
			}
		}
	}
	return found
}

func owned(found place, path string) models.LineEntry {
	at := models.LinePlace{Pull: found.pull, Line: found.line}
	if found.path != path {
		at.Path = found.path
	}
	return models.LineEntry{Owner: &at}
}

func unchanged(found trace, path string) models.LineEntry {
	if found.owner != nil {
		return owned(*found.owner, path)
	}
	candidates := []models.LinePlace{}
	for _, t := range found.shown {
		candidates = append(candidates, *owned(t, path).Owner)
	}
	return models.LineEntry{Candidates: candidates}
}

func touching(steps []Step, chunk diff.Chunk) []int {
	pulls := []int{}
	for _, s := range steps {
		var path string
		var has bool
		if chunk.Status == "D" {
			path = chunk.OldPath
			if moved, ok := s.FromBase.ByOld[chunk.OldPath]; ok {
				path = moved.Path
			}
			_, has = s.Own.ByOld[path]
		} else {
			path = chunk.Path
			if moved, ok := s.FromTop.ByNew[chunk.Path]; ok {
				path = moved.OldPath
				if moved.Status == "A" {
					path = ""
				}
			}
			_, has = s.Own.ByNew[path]
		}
		if path != "" && has {
			pulls = append(pulls, s.Pull)
		}
	}
	return pulls
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
