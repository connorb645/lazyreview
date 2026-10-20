package review

import (
	"reflect"
	"testing"

	"github.com/connorb645/lazyreview/internal/models"
)

func remark(id int, login string) models.GitHubThreadRemark {
	r := models.GitHubThreadRemark{DatabaseID: id, Body: "Comment", CreatedAt: "2026-10-15T10:00:00Z", DiffHunk: "@@ -1,5 +1,5 @@\n a\n b\n-c\n+C\n d\n e"}
	if login != "" {
		r.Author = &models.GitHubAuthor{Login: login, Typename: "User"}
	}
	return r
}

func thread(id, path string, line int) models.GitHubReviewThread {
	return models.GitHubReviewThread{ID: id, Path: path, Line: &line, DiffSide: "RIGHT", SubjectType: "LINE",
		Comments: models.GitHubThreadComments{TotalCount: 1, Nodes: []models.GitHubThreadRemark{remark(11, "alex")}}}
}

func TestAThreadIsReadWithItsStateItsCommentsAndWhoWroteEach(t *testing.T) {
	start, left := 2, "LEFT"
	node := thread("T1", "docs/guide.md", 3)
	node.IsResolved, node.ResolvedBy, node.StartLine, node.StartDiffSide = true, &models.GitHubUser{Login: "sam"}, &start, &left
	bot := remark(12, "lint-helper")
	bot.Author.Typename = "Bot"
	node.Comments = models.GitHubThreadComments{TotalCount: 104, Nodes: []models.GitHubThreadRemark{remark(11, "alex"), bot, remark(13, "")}}
	got := ReadThreads(30, []models.GitHubReviewThread{node})[0]
	if got.StartLine != 2 || got.StartSide != "LEFT" || got.ResolvedBy != "sam" || got.More != 101 || got.Hunk != "-c\n+C\n d\n e" {
		t.Errorf("the thread reads as %+v", got)
	}
	var authors [][2]any
	for _, c := range got.Comments {
		authors = append(authors, [2]any{c.Author, c.Bot})
	}
	if want := [][2]any{{"alex", false}, {"lint-helper", true}, {"ghost", false}}; !reflect.DeepEqual(authors, want) {
		t.Errorf("the comments' authors read as %v", authors)
	}
}

func TestOnAStackAThreadSitsOnTheCombinedLineThatShowsItsOwnLine(t *testing.T) {
	stack := &models.StackRecord{Lines: map[string]map[string]models.LineMap{
		"web/cart.ts": {
			"RIGHT": {"4": {Owner: &models.LinePlace{Pull: 21, Line: 2}}, "5": {Owner: &models.LinePlace{Pull: 21, Line: 3}},
				"9": {Candidates: []models.LinePlace{{Pull: 21, Line: 7}, {Pull: 22, Line: 8}}}},
			"LEFT": {"6": {Owner: &models.LinePlace{Pull: 21, Line: 6}}},
		},
		"web/total.ts": {"RIGHT": {"3": {Owner: &models.LinePlace{Pull: 22, Line: 3, Path: "web/sum.ts"}}}},
	}}
	ranged := thread("T1", "web/cart.ts", 3)
	two := 2
	ranged.StartLine = &two
	removed := thread("T3", "web/cart.ts", 6)
	removed.DiffSide = "LEFT"
	threads := append(ReadThreads(21, []models.GitHubReviewThread{ranged}), ReadThreads(22, []models.GitHubReviewThread{thread("T2", "web/cart.ts", 8)})...)
	threads = append(threads, ReadThreads(21, []models.GitHubReviewThread{removed, thread("T5", "web/cart.ts", 12)})...)
	threads = append(threads, ReadThreads(22, []models.GitHubReviewThread{thread("T4", "web/sum.ts", 3)})...)
	var at []*models.Lines
	for _, t := range PlaceThreads(threads, stack) {
		at = append(at, t.At)
	}
	want := []*models.Lines{
		{Path: "web/cart.ts", Side: "RIGHT", Line: 5, StartLine: 4, StartSide: "RIGHT"},
		{Path: "web/cart.ts", Side: "RIGHT", Line: 9},
		{Path: "web/cart.ts", Side: "LEFT", Line: 6},
		nil,
		{Path: "web/total.ts", Side: "RIGHT", Line: 3},
	}
	if !reflect.DeepEqual(at, want) {
		t.Errorf("the threads sit at %v", at)
	}
}
