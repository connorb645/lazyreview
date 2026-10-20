package review

import (
	"regexp"
	"sort"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var markdownPath = regexp.MustCompile(`(?i)\.(md|markdown)$`)

// FileView is one file of a review's page, ready to draw, and false for a
// path the page does not draw: its diff hunk by hunk, each row on a stack
// with the pull request that changed it and the words saying where a comment
// on it goes; the runs of unchanged lines between the hunks, for a changed
// file whose whole text the page kept; its whole text at the head; and for a
// Markdown file with a diff, its Rendered view, each block with the pull
// requests that changed its lines. An unchanged file of the context has
// only its text.
func FileView(tr *i18n.TranslationSet, page *models.PageRecord, path string) (models.File, bool) {
	file, ok := fileOf(page, path)
	if !ok {
		for _, f := range page.Model.ContextFiles {
			if f.Path == path {
				return models.File{Path: path, Hunks: []models.Hunk{}, Gaps: []models.Gap{}, Whole: WholeText(page, path)}, true
			}
		}
		return models.File{}, false
	}
	hunks := ReadHunks(page.Diff[path])
	if hunks == nil {
		hunks = []models.Hunk{}
	}
	for h := range hunks {
		for r := range hunks[h].Rows {
			hunks[h].Rows[r].Owner, hunks[h].Rows[r].Words = mark(tr, page.Stack, path, hunks[h].Rows[r])
		}
	}
	whole := WholeText(page, path)
	view := models.File{Path: path, Hunks: hunks, Gaps: []models.Gap{}, Whole: whole}
	if markdownPath.MatchString(path) && len(hunks) > 0 {
		rendered := withPulls(MarkdownBlocks(file, hunks, whole), hunks)
		view.Rendered = &rendered
	}
	if file.Status == "M" || file.Status == "R" {
		view.Gaps = gaps(hunks, whole)
	}
	return view, true
}

func mark(tr *i18n.TranslationSet, stack *models.StackRecord, path string, row models.Row) (int, string) {
	if stack == nil || row.Kind == "note" {
		return 0, ""
	}
	entry := LineEntry(tr, stack.Lines, path, row.New, "RIGHT")
	if row.Kind == "del" {
		entry = LineEntry(tr, stack.Lines, path, row.Old, "LEFT")
	}
	changed := row.Kind != "same"
	owner := 0
	if changed && entry.Pull != 0 {
		owner = entry.Pull
	}
	return owner, placeWords(tr, entry, changed, path)
}

func placeWords(tr *i18n.TranslationSet, entry StackPlace, changed bool, path string) string {
	at := func(p StackPlace) string {
		words := tr.StackPlace
		if p.Path != path {
			words = tr.StackPlaceOf
		}
		return utils.ResolvePlaceholderString(words, map[string]string{"pull": utils.Itoa(p.Pull), "line": utils.Itoa(p.Line), "path": p.Path})
	}
	switch {
	case entry.Pull != 0 && changed:
		return utils.ResolvePlaceholderString(tr.ChangedIn, map[string]string{"pull": utils.Itoa(entry.Pull), "place": at(entry)})
	case entry.Pull != 0:
		return utils.ResolvePlaceholderString(tr.LastChanged, map[string]string{"pull": utils.Itoa(entry.Pull), "place": at(entry)})
	case entry.None != "":
		return utils.ResolvePlaceholderString(tr.NoPullTakes, map[string]string{"why": entry.None})
	case len(entry.Candidates) == 0:
		return tr.NoneShows
	case len(entry.Candidates) == 1:
		return utils.ResolvePlaceholderString(tr.OneShows, map[string]string{"place": at(entry.Candidates[0])})
	}
	places := make([]string, len(entry.Candidates))
	for i, c := range entry.Candidates {
		places[i] = at(c)
	}
	return utils.ResolvePlaceholderString(tr.SeveralShow, map[string]string{"places": joinWith(places, tr.And)})
}

func withPulls(rendered models.Rendered, hunks []models.Hunk) models.Rendered {
	var owned []models.Row
	for _, h := range hunks {
		for _, r := range h.Rows {
			if r.Owner != 0 {
				owned = append(owned, r)
			}
		}
	}
	for i, b := range rendered.Blocks {
		seen := map[int]bool{}
		pulls := []int{}
		for _, r := range owned {
			line := r.New
			if b.State == "del" {
				line = r.Old
			}
			if r.Kind == b.State && line >= b.StartLine && line <= b.Line && !seen[r.Owner] {
				seen[r.Owner] = true
				pulls = append(pulls, r.Owner)
			}
		}
		sort.Ints(pulls)
		rendered.Blocks[i].Pulls = pulls
	}
	return rendered
}

func gaps(hunks []models.Hunk, whole []string) []models.Gap {
	out := []models.Gap{}
	if whole == nil || len(hunks) == 0 {
		return out
	}
	next := 1
	for i, h := range hunks {
		if h.NewStart > next {
			out = append(out, models.Gap{Before: i, From: next, To: h.NewStart - 1, Delta: h.OldStart - h.NewStart})
		}
		next = h.NewStart + h.NewCount
	}
	last := hunks[len(hunks)-1]
	if next <= len(whole) {
		out = append(out, models.Gap{Before: len(hunks), From: next, To: len(whole), Delta: last.OldStart + last.OldCount - (last.NewStart + last.NewCount)})
	}
	return out
}

func joinWith(words []string, and string) string {
	out := ""
	for i, w := range words {
		if i > 0 {
			out += " " + and + " "
		}
		out += w
	}
	return out
}
