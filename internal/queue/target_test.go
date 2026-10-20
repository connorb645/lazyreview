package queue

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/models"
)

func targetOn(t *testing.T, rev models.Review, page *models.PageRecord, path string, line, start int, startSide string, pull int) models.Where {
	t.Helper()
	return Target(tr, rev, page, models.Lines{Path: path, Side: "RIGHT", Line: line, StartLine: start, StartSide: startSide}, pull)
}

func TestACommentOnASinglePullRequestGoesWhereItWasMade(t *testing.T) {
	rev := models.Review{Members: []models.ReviewMember{member(7)}}
	page := &models.PageRecord{Model: models.Model{Files: []models.FileEntry{{Path: "a.ts"}}},
		Diff: map[string]string{"a.ts": "diff --git a/a.ts b/a.ts\n--- a/a.ts\n+++ b/a.ts\n@@ -1,3 +1,4 @@\n one\n+two\n three\n four\n@@ -20,2 +21,2 @@\n-x\n+y\n z"}}
	at := func(line, start int, startSide string, pull int) models.Where {
		return targetOn(t, rev, page, "a.ts", line, start, startSide, pull)
	}
	if w := at(3, 2, "", 0); w.Goes == nil || *w.Goes != (models.Target{Pull: 7, Path: "a.ts", Side: "RIGHT", Line: 3, StartLine: 2, StartSide: "RIGHT"}) || w.Words != "Goes on #7, lines 2-3." {
		t.Errorf("a range goes %+v, %q", w.Goes, w.Words)
	}
	for _, c := range []struct {
		w    models.Where
		want string
	}{
		{at(21, 20, "LEFT", 0), "Goes on #7, removed line 20 to line 21."},
		{at(9, 0, "", 0), "GitHub takes comments only on lines of the diff, and this one is not. Ask Claude about it instead."},
		{at(1, 0, "", 0), "Goes on #7, line 1."},
		{at(21, 2, "", 0), "GitHub takes a range only inside one hunk"},
		{at(2, 3, "", 0), "A range runs from its first line to its last"},
		{at(2, 0, "", 8), "This review is #7's, not #8's."},
		{targetOn(t, rev, page, "b.ts", 1, 0, "", 0), "This file is not in the change, so a comment cannot go on it. Ask Claude about it instead."},
		{targetOn(t, models.Review{Members: []models.ReviewMember{{Pull: 7, State: "closed"}}}, page, "a.ts", 2, 0, "", 0), "#7 is closed, so nothing can be queued for it."},
		{targetOn(t, models.Review{Members: rev.Members, Moved: &models.Moved{To: "x"}}, page, "a.ts", 2, 0, "", 0), "This review moved"},
		{targetOn(t, models.Review{Kind: "branch"}, page, "a.ts", 2, 0, "", 0), "A branch with no pull request takes no comments"},
	} {
		if got := c.w.Words + c.w.Refused; !strings.HasPrefix(got, c.want) {
			t.Errorf("%q, not %q", got, c.want)
		}
	}
}

func TestOnAStackACommentGoesToThePullRequestThatLastChangedItsLineOrNamesOneOfThoseThatShowIt(t *testing.T) {
	rev := models.Review{Members: []models.ReviewMember{member(7), member(8)}}
	var lines map[string]map[string]models.LineMap
	if err := json.Unmarshal([]byte(`{"a.ts": {"RIGHT": {"1": {"c": [[7, 1], [8, 1]]}, "2": [8, 2], "3": {"c": [[8, 3]]}, "4": {"c": []}, "21": {"x": "no pull request's own diff adds it"}, "22": [7, 30, "old/a.ts"]}, "LEFT": {}}}`), &lines); err != nil {
		t.Fatal(err)
	}
	stack := &models.StackRecord{Members: []models.StackMember{{Pull: 7, Head: rev.Members[0].Head}, {Pull: 8, Head: rev.Members[1].Head}}, Lines: lines}
	page := &models.PageRecord{Model: models.Model{Files: []models.FileEntry{{Path: "a.ts"}}},
		Diff: map[string]string{"a.ts": "@@ -1,3 +1,4 @@\n one\n+two\n three\n four\n@@ -20,2 +21,2 @@\n-x\n+y\n z"}, Stack: stack}
	at := func(line, start, pull int) models.Where { return targetOn(t, rev, page, "a.ts", line, start, "", pull) }
	if w := at(2, 0, 0); w.Goes == nil || *w.Goes != (models.Target{Pull: 8, Path: "a.ts", Side: "RIGHT", Line: 2}) || w.Words != "Goes on #8, line 2." {
		t.Errorf("a changed line goes %+v, %q", w.Goes, w.Words)
	}
	if w := at(1, 0, 0); len(w.Choose) != 2 || w.Choose[0] != (models.Choice{Pull: 7, Path: "a.ts", Line: 1}) || w.Words != "Unchanged, and shown by #7 and #8. Choose the pull request it goes on." {
		t.Errorf("an unchanged line shown by two gives %+v", w)
	}
	for _, c := range []struct {
		w    models.Where
		want string
	}{
		{at(1, 0, 8), "Goes on #8, line 1."},
		{at(2, 0, 7), "#8 last changed this line, not #7."},
		{at(2, 1, 0), "Goes on #8, lines 1-2."},
		{at(4, 0, 0), "Unchanged, and no pull request in the stack shows this line, so it takes no comment."},
		{at(21, 0, 0), "No pull request in the stack can take a comment here: no pull request's own diff adds it."},
		{at(22, 0, 0), "Goes on #7, line 30 of old/a.ts."},
		{at(3, 2, 0), "Goes on #8, lines 2-3."},
		{at(3, 1, 7), "#7's diff does not show this line."},
	} {
		if got := c.w.Words + c.w.Refused; got != c.want {
			t.Errorf("%q, not %q", got, c.want)
		}
	}
	merged := models.Review{Members: []models.ReviewMember{member(7), member(8)}}
	merged.Members[1].State = "merged"
	if w := targetOn(t, merged, page, "a.ts", 2, 0, "", 0); w.Refused != "#8 is merged, so nothing can be queued for it." {
		t.Errorf("a merged member gives %q", w.Refused)
	}
	moved := *page
	moved.Stack = &models.StackRecord{Members: []models.StackMember{{Pull: 7, Head: "other"}, {Pull: 8, Head: rev.Members[1].Head}}, Lines: lines}
	if w := targetOn(t, rev, &moved, "a.ts", 2, 0, "", 0); !strings.HasPrefix(w.Refused, "This page shows other heads of #7 than the review holds") {
		t.Errorf("a page of other heads gives %q", w.Refused)
	}
}
