package queue

import (
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/review"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// Target is where a comment on lines the app shows goes, or why none can:
// the pull request that takes it, at its own path and lines, the ones
// GitHub takes for its diff, in Goes; the pull requests of a stack that
// each show an unchanged line, one of which the comment must name, in
// Choose; or why not, in Refused. Words say it as a sentence the app shows
// as it is. The lines must be lines of the change's diff, and a range must
// stay inside one hunk of it. On a stack, the line is traced through the
// stack's line map to the pull request that last changed it; pull names one
// pull request where an unchanged line of a stack shows in several.
func Target(tr *i18n.TranslationSet, rev models.Review, page *models.PageRecord, at models.Lines, pull int) models.Where {
	if rev.IsBranch() {
		return refused(tr.BranchTakesNone)
	}
	if rev.Moved != nil {
		return refused(tr.ReviewMovedHere)
	}
	inChange := false
	for _, f := range page.Model.Files {
		if f.Path == at.Path {
			inChange = true
		}
	}
	if !inChange {
		return refused(tr.NotInChange)
	}
	startSide := ""
	if at.StartLine != 0 {
		startSide = at.StartSide
		if startSide == "" {
			startSide = at.Side
		}
	}
	places := DiffPlaces(page.Diff[at.Path])
	end, ok := places[at.Side][at.Line]
	if !ok {
		return refused(tr.NotADiffLine)
	}
	if at.StartLine != 0 {
		start, ok := places[startSide][at.StartLine]
		if !ok || start.Hunk != end.Hunk {
			return refused(tr.LeavesHunk)
		}
		if start.Row >= end.Row {
			return refused(tr.Backwards)
		}
	}
	if page.Stack == nil {
		member := rev.Members[0]
		if pull != 0 && pull != member.Pull {
			return refused(utils.ResolvePlaceholderString(tr.NotThisReview, map[string]string{"own": strconv.Itoa(member.Pull), "given": strconv.Itoa(pull)}))
		}
		return goes(tr, rev, at.Path, models.Target{Pull: member.Pull, Path: at.Path, Side: at.Side, Line: at.Line, StartLine: at.StartLine, StartSide: startSide})
	}
	return onStack(tr, rev, page.Stack, at, pull, startSide)
}

type taking struct {
	end   review.StackPlace
	start *review.StackPlace
}

func onStack(tr *i18n.TranslationSet, rev models.Review, stack *models.StackRecord, at models.Lines, pull int, startSide string) models.Where {
	heads := map[int]string{}
	for _, m := range stack.Members {
		heads[m.Pull] = m.Head
	}
	var out []int
	for _, m := range rev.Members {
		if heads[m.Pull] != m.Head {
			out = append(out, m.Pull)
		}
	}
	if len(out) > 0 {
		return refused(utils.ResolvePlaceholderString(tr.OtherHeads, map[string]string{"pulls": pullList(out, ", ")}))
	}
	end := review.LineEntry(tr, stack.Lines, at.Path, at.Line, at.Side)
	if end.None != "" {
		return refused(utils.ResolvePlaceholderString(tr.NoPullTakes, map[string]string{"why": end.None}))
	}
	ends := end.Candidates
	if end.Pull != 0 {
		ends = []review.StackPlace{end}
	}
	if len(ends) == 0 {
		return refused(tr.NoneShows)
	}
	var start *review.StackPlace
	if at.StartLine != 0 {
		s := review.LineEntry(tr, stack.Lines, at.Path, at.StartLine, startSide)
		start = &s
	}
	var takes []taking
	for _, e := range ends {
		t := taking{end: e}
		if start != nil {
			t.start = startOn(*start, e)
			if t.start == nil {
				continue
			}
		}
		takes = append(takes, t)
	}
	if pull != 0 {
		for _, t := range takes {
			if t.end.Pull == pull {
				return goes(tr, rev, at.Path, targetOf(t, at.Side, startSide))
			}
		}
		if end.Pull != 0 && end.Pull != pull {
			return refused(utils.ResolvePlaceholderString(tr.LastChangedBy, map[string]string{"last": strconv.Itoa(end.Pull), "given": strconv.Itoa(pull)}))
		}
		shown := false
		for _, e := range ends {
			if e.Pull == pull {
				shown = true
			}
		}
		if !shown {
			return refused(utils.ResolvePlaceholderString(tr.NotShownBy, map[string]string{"pull": strconv.Itoa(pull)}))
		}
		return refused(utils.ResolvePlaceholderString(tr.NotAllTo, map[string]string{"pull": strconv.Itoa(pull)}))
	}
	switch len(takes) {
	case 0:
		return refused(tr.NotAllToOne)
	case 1:
		return goes(tr, rev, at.Path, targetOf(takes[0], at.Side, startSide))
	}
	where := models.Where{}
	var numbers []int
	for _, t := range takes {
		where.Choose = append(where.Choose, models.Choice{Pull: t.end.Pull, Path: t.end.Path, Line: t.end.Line})
		numbers = append(numbers, t.end.Pull)
	}
	where.Words = utils.ResolvePlaceholderString(tr.ChooseAmong, map[string]string{"pulls": pullList(numbers, " "+tr.And+" ")})
	return where
}

func startOn(start, end review.StackPlace) *review.StackPlace {
	found := &start
	if start.Pull == 0 {
		found = nil
		for i := range start.Candidates {
			if start.Candidates[i].Pull == end.Pull {
				found = &start.Candidates[i]
				break
			}
		}
	}
	if found != nil && found.Pull == end.Pull && found.Path == end.Path {
		return found
	}
	return nil
}

func targetOf(t taking, side, startSide string) models.Target {
	target := models.Target{Pull: t.end.Pull, Path: t.end.Path, Side: side, Line: t.end.Line}
	if t.start != nil {
		target.StartLine, target.StartSide = t.start.Line, startSide
	}
	return target
}

func goes(tr *i18n.TranslationSet, rev models.Review, shownPath string, place models.Target) models.Where {
	var member *models.ReviewMember
	for i := range rev.Members {
		if rev.Members[i].Pull == place.Pull {
			member = &rev.Members[i]
		}
	}
	if member == nil {
		return refused(utils.ResolvePlaceholderString(tr.LeftStack, map[string]string{"pull": strconv.Itoa(place.Pull)}))
	}
	if member.State != "" && member.State != "open" {
		return refused(utils.ResolvePlaceholderString(tr.PullEndedHere, map[string]string{"pull": strconv.Itoa(place.Pull), "state": member.State}))
	}
	words := map[string]string{"pull": strconv.Itoa(place.Pull), "lines": LinesWords(tr, place), "path": place.Path}
	sentence := tr.GoesOn
	if place.Path != shownPath {
		sentence = tr.GoesOnOf
	}
	return models.Where{Goes: &place, Words: utils.ResolvePlaceholderString(sentence, words)}
}

// LinesWords is lines in words, such as "line 3", "removed lines 7-9" or
// "removed line 3 to line 5".
func LinesWords(tr *i18n.TranslationSet, t models.Target) string {
	side := func(s string) string {
		if s == "LEFT" {
			return tr.RemovedLineWord
		}
		return tr.LineWord
	}
	startSide := t.StartSide
	if startSide == "" {
		startSide = t.Side
	}
	words := map[string]string{"side": side(t.Side), "line": strconv.Itoa(t.Line), "startSide": side(startSide), "start": strconv.Itoa(t.StartLine)}
	switch {
	case t.StartLine == 0:
		return utils.ResolvePlaceholderString(tr.OneLine, words)
	case startSide != t.Side:
		return utils.ResolvePlaceholderString(tr.LineToLine2, words)
	}
	return utils.ResolvePlaceholderString(tr.LinesRange, words)
}

func refused(words string) models.Where {
	return models.Where{Refused: words}
}

func pullList(numbers []int, joiner string) string {
	parts := make([]string, len(numbers))
	for i, n := range numbers {
		parts[i] = "#" + strconv.Itoa(n)
	}
	return strings.Join(parts, joiner)
}
