package queue

import (
	"reflect"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
)

func regroupMember(pull int, colour int, comments ...models.Comment) models.ReviewMember {
	head := strings.Repeat(string(rune('0'+pull)), 10)
	return models.ReviewMember{Pull: pull, Head: head, Start: "start000", Author: "author", State: "open", Color: colour, Comments: comments}
}

func regroupComment(id int, side string) models.Comment {
	return models.Comment{ID: id, Path: "a.ts", Line: 3, Side: side, Body: "Comment"}
}

func TestRegroupingKeepsCommentsAndColoursAndMarksStaleWhatMayHaveMoved(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	previous := &models.Review{Members: []models.ReviewMember{
		regroupMember(1, 3, regroupComment(1, "RIGHT")),
		regroupMember(2, 1, regroupComment(2, "LEFT"), regroupComment(3, "RIGHT")),
	}}
	fresh := []models.ReviewMember{regroupMember(1, 0), regroupMember(2, 0), regroupMember(4, 0)}
	fresh[0].Head = "moved1head"
	fresh[1].Start = "start111"
	out, err := Regroup(tr, Regrouping{Previous: previous, Fresh: fresh, Anchor: "github.com/o/r/1", Now: "t"})
	if err != nil {
		t.Fatal(err)
	}
	var colours []int
	for _, m := range out.Members {
		colours = append(colours, m.Color)
	}
	if !reflect.DeepEqual(colours, []int{3, 1, 2}) {
		t.Errorf("a new member takes the first free colour, not %v", colours)
	}
	if !strings.HasPrefix(out.Members[0].Comments[0].Stale, "queued at 11111111, and #1 is now at moved1he") {
		t.Errorf("a moved head marks the comment stale, not %q", out.Members[0].Comments[0].Stale)
	}
	if !strings.Contains(out.Members[1].Comments[0].Stale, "queued on a removed line, and #2's diff now starts from start111") {
		t.Errorf("a moved start marks a removed line stale, not %q", out.Members[1].Comments[0].Stale)
	}
	if out.Members[1].Comments[1].Stale != "" || len(out.Notes) != 2 {
		t.Errorf("a comment on the new side stays when only the start moved: %+v, %q", out.Members[1].Comments[1], out.Notes)
	}
}

func TestRegroupingMovesAnotherQueueInAndRefusesToDropAnOpenMemberWithComments(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	previous := &models.Review{Members: []models.ReviewMember{regroupMember(1, 1, regroupComment(4, "RIGHT"))}}
	other := models.Review{ID: "github.com/o/r/2", Members: []models.ReviewMember{regroupMember(2, 1, regroupComment(1, "RIGHT"), regroupComment(2, "RIGHT"))}}
	ref, err := models.ParseReviewRef(tr, "github.com/o/r/2")
	if err != nil {
		t.Fatal(err)
	}
	out, err := Regroup(tr, Regrouping{Previous: previous, Fresh: []models.ReviewMember{regroupMember(1, 0), regroupMember(2, 0)},
		Absorbed: []Absorbed{{Ref: ref, Review: other}}, Anchor: "github.com/o/r/1", Now: "now"})
	if err != nil {
		t.Fatal(err)
	}
	if ids := []int{out.Members[1].Comments[0].ID, out.Members[1].Comments[1].ID}; !reflect.DeepEqual(ids, []int{5, 6}) {
		t.Errorf("the moved comments are numbered %v", ids)
	}
	if moved := out.Moved[0].Review; !reflect.DeepEqual(moved.Moved, &models.Moved{To: "github.com/o/r/1", Stack: []int{1, 2}, At: "now"}) || len(moved.Members[0].Comments) != 0 {
		t.Errorf("the absorbed review is left as %+v", moved)
	}
	stacked := &models.Review{Members: out.Members, Earlier: out.Earlier}
	_, err = Regroup(tr, Regrouping{Previous: stacked, Fresh: []models.ReviewMember{regroupMember(1, 0)}, Leaving: map[int]models.ReviewMember{2: {State: "open"}}})
	if err == nil || !strings.Contains(err.Error(), "#2 is still open with 2 queued comments, so it cannot leave the stack") {
		t.Errorf("an open member with comments leaves the stack: %v", err)
	}
	emptied := out.Members[1]
	emptied.Comments = nil
	then := "then"
	left, err := Regroup(tr, Regrouping{Previous: &models.Review{Members: []models.ReviewMember{out.Members[0], emptied}}, Fresh: []models.ReviewMember{regroupMember(1, 0)},
		Leaving: map[int]models.ReviewMember{2: {State: "closed", EndedAt: &then}}})
	if err != nil || len(left.Earlier) != 1 || left.Earlier[0].Pull != 2 || left.Earlier[0].State != "closed" || *left.Earlier[0].EndedAt != "then" {
		t.Errorf("a closed member leaves the stack as %+v, %v", left.Earlier, err)
	}
}
