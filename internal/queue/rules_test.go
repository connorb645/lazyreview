package queue

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
)

var tr = i18n.EnglishTranslationSet()

func member(pull int, comments ...models.Comment) models.ReviewMember {
	head := strings.Repeat(fmt.Sprint(pull), 10)[:10]
	return models.ReviewMember{Pull: pull, Head: head, Start: "start000", Author: "author", State: "open", Comments: comments}
}

func comment(id int) models.Comment {
	return models.Comment{ID: id, Path: "a.ts", Line: 3, Side: "RIGHT", Body: fmt.Sprintf("Comment %d", id)}
}

func TestASetIDNamesOneActionOnOneHeadWithOneSetOfComments(t *testing.T) {
	review := models.Review{Repo: "octo/widgets", Me: "reviewer"}
	m := member(12, comment(1))
	id := SetID(review, m, "comment")
	if !regexp.MustCompile(`^[0-9a-f]{12}$`).MatchString(id) {
		t.Fatalf("the set id is %q", id)
	}
	changedBody, quoted, moved := m, m, m
	changedBody.Comments = []models.Comment{{ID: 1, Path: "a.ts", Line: 3, Side: "RIGHT", Body: "Changed"}}
	quoted.Comments = []models.Comment{{ID: 1, Path: "a.ts", Line: 3, Side: "RIGHT", Body: "Comment 1", Code: "shown only"}}
	moved.Head = "other"
	empty := m
	empty.Comments = nil
	for _, c := range []struct {
		same bool
		got  string
		why  string
	}{
		{true, SetID(review, m, "comment"), "the same queue gives the same set"},
		{false, SetID(review, m, "approve-with-comments"), "another action"},
		{false, SetID(review, moved, "comment"), "another head"},
		{false, SetID(review, changedBody, "comment"), "other words"},
		{true, SetID(review, quoted, "comment"), "what the page shows beside a comment is not part of what is posted"},
	} {
		if (c.got == id) != c.same {
			t.Errorf("%s: %s against %s", c.why, c.got, id)
		}
	}
	if SetID(review, empty, "approve") != SetID(review, m, "approve") {
		t.Errorf("Approve sends no comments, so they change its set")
	}
}

func TestTheActionsOfferedLeaveOutAStaleCommentAndOnesOwnApproval(t *testing.T) {
	review := models.Review{Repo: "o/r", Me: "reviewer"}
	stale := comment(2)
	stale.Stale = "moved"
	own := member(1, comment(1))
	own.Author = "reviewer"
	for _, c := range []struct {
		m    models.ReviewMember
		want string
	}{
		{member(1), "approve"},
		{member(1, comment(1)), "comment approve-with-comments approve"},
		{member(1, comment(1), stale), "approve"},
		{own, "comment"},
	} {
		if got := strings.Join(Offered(review, c.m), " "); got != c.want {
			t.Errorf("offered %q, not %q", got, c.want)
		}
	}
}

func keys(places map[int]Place) []int {
	var out []int
	for k := range places {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func TestTheLinesOfAPatchArePlacedOnBothSidesEachInItsHunkAndItsRow(t *testing.T) {
	places := DiffPlaces("@@ -3,3 +3,4 @@\n one\n-two\n+TWO\n+three\n four\n\\ No newline at end of file\n@@ -20 +21 @@\n-x\n+y")
	if fmt.Sprint(keys(places["RIGHT"])) != "[3 4 5 6 21]" || fmt.Sprint(keys(places["LEFT"])) != "[3 4 5 20]" {
		t.Fatalf("the lines are %v and %v", keys(places["RIGHT"]), keys(places["LEFT"]))
	}
	if places["LEFT"][3] != (Place{0, 1}) || places["RIGHT"][3] != (Place{0, 1}) || places["RIGHT"][21] != (Place{1, 9}) {
		t.Errorf("the places are %v", places)
	}
	if len(DiffPlaces("")["RIGHT"]) != 0 {
		t.Errorf("a file GitHub shows no patch for has lines")
	}
	if fmt.Sprint(keys(DiffPlaces("@@ -1 +1 @@\n-a\n+b\n")["RIGHT"])) != "[1]" {
		t.Errorf("a trailing newline adds a line")
	}
}

func TestGitHubTakesACommentOnALineOfTheDiffAndARangeOnlyInsideOneHunk(t *testing.T) {
	places := DiffPlaces("@@ -3,3 +3,4 @@\n one\n-two\n+TWO\n+three\n four\n@@ -20,2 +21,2 @@\n-x\n+y\n z")
	at := func(line int, side string, start int, startSide string) string {
		return PlaceProblem(tr, places, models.Comment{Line: line, Side: side, StartLine: start, StartSide: startSide})
	}
	for _, c := range []struct{ got, want string }{
		{at(4, "RIGHT", 0, ""), ""},
		{at(4, "LEFT", 0, ""), ""},
		{at(9, "RIGHT", 0, ""), "is not on a line of the diff"},
		{at(6, "RIGHT", 3, ""), ""},
		{at(5, "RIGHT", 4, "LEFT"), ""},
		{at(22, "RIGHT", 4, ""), "runs from line 4 across more than one hunk of the diff, and GitHub takes a range only inside one hunk"},
		{at(5, "RIGHT", 8, ""), "starts on line 8, which is not on a line of the diff"},
		{at(4, "RIGHT", 5, ""), "starts on line 5, which does not come before its last line in the diff"},
		{at(4, "LEFT", 4, "RIGHT"), "starts on line 4, which does not come before its last line in the diff"},
	} {
		if c.got != c.want {
			t.Errorf("%q, not %q", c.got, c.want)
		}
	}
	if f := Posted(models.Comment{Path: "a.ts", Line: 6, Side: "RIGHT", StartLine: 3, Body: "b"}); f.StartSide != "RIGHT" {
		t.Errorf("a range queued before ranges had a side starts on %q", f.StartSide)
	}
}

func TestTheQueueViewOffersNoButtonForAnEndedPullRequestOneThatLeftOrAMovedReview(t *testing.T) {
	merged := member(2)
	merged.State = "merged"
	review := models.Review{Repo: "o/r", Me: "me", Members: []models.ReviewMember{member(1, comment(1)), merged}, Earlier: []models.ReviewMember{member(3)}}
	sets := func(m models.QueueMember) string {
		var out []string
		for _, a := range Actions {
			if _, ok := m.Sets[a]; ok {
				out = append(out, a)
			}
		}
		return strings.Join(out, " ")
	}
	view := View(tr, review)
	if sets(view.Members[0]) != "comment approve-with-comments approve" || sets(view.Members[1]) != "" || sets(view.Earlier[0]) != "" {
		t.Errorf("the buttons are %q, %q and %q", sets(view.Members[0]), sets(view.Members[1]), sets(view.Earlier[0]))
	}
	moved := review
	moved.Moved = &models.Moved{To: "x"}
	if v := View(tr, moved); sets(v.Members[0]) != "" {
		t.Errorf("a moved review offers %q", sets(v.Members[0]))
	}
	for action, want := range map[string]string{
		"comment":               "This comment goes on o/r#1 under your name, exactly as shown.",
		"approve-with-comments": "You approve o/r#1, and this comment goes with the approval under your name, exactly as shown.",
		"approve":               "You approve o/r#1 with no comments. The queued 1 comment stays queued, unposted.",
	} {
		if got := view.Members[0].Sets[action].Words; got != want {
			t.Errorf("%s says %q", action, got)
		}
	}
	if got := PressWords(tr, review, member(1, comment(1), comment(2)), "comment"); !strings.HasPrefix(got, "These 2 comments go on o/r#1") {
		t.Errorf("two comments say %q", got)
	}
	if _, err := LiveMember(tr, &review, 3); err == nil || err.Error() != "#3 is not in this stack any more" {
		t.Errorf("a member that left gives %v", err)
	}
	if _, err := LiveMember(tr, &models.Review{Moved: &models.Moved{To: "github.com/o/r/9"}}, 1); err == nil || err.Error() != "this review moved to github.com/o/r/9; nothing changed here" {
		t.Errorf("a moved review gives %v", err)
	}
}
