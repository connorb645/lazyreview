package queue

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/commands/github"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/testsupport"
)

const reviewURL = "https://github.com/octo/widgets/pull/30#pullrequestreview-7"

var (
	single = models.PullRef{Host: "github.com", Repo: "octo/widgets", Pull: 30}
	stack  = models.PullRef{Host: "github.com", Repo: "octo/widgets", Pull: 21}
)

// standIn is the queue of the stand-in data home the recorded answers come
// from: #30 alone, with comment 1 queued on line 3 of docs/guide.md, and
// #21 and #22 as a stack, each with one comment queued; and the fake GitHub
// that answers for them, as the account reviewer.
type standIn struct {
	t     *testing.T
	queue *Service
	store *store.ReviewStore
	gh    *testsupport.FakeGh
}

func newStandIn(t *testing.T) standIn {
	t.Helper()
	gh := testsupport.FakeGhFrom(t, testsupport.Testdata("github.json"))
	reviews := store.NewReviewStore(tr, store.DataHome{Root: testsupport.StandInHome(t)}, store.NewLocks(tr))
	q := NewService(tr, reviews, func(host, account string) *github.GitHub { return github.New(tr, host, account, gh.Run) }, func() string { return "2026-10-15T13:00:00.000Z" })
	return standIn{t: t, queue: q, store: reviews, gh: gh}
}

func (s standIn) build(ref models.ReviewRef) string {
	build, err := s.store.PageBuild(ref)
	if err != nil {
		s.t.Fatal(err)
	}
	return build
}

func (s standIn) review(ref models.ReviewRef) models.Review {
	r, err := s.store.Read(ref)
	if err != nil || r == nil {
		s.t.Fatalf("could not read %s: %v", ref.ID(), err)
	}
	return *r
}

func (s standIn) member(ref models.ReviewRef, pull int) models.QueueMember {
	for _, m := range View(tr, s.review(ref)).Members {
		if m.Pull == pull {
			return m
		}
	}
	s.t.Fatalf("no member %d", pull)
	return models.QueueMember{}
}

func (s standIn) version(ref models.ReviewRef, id int) string {
	for _, m := range View(tr, s.review(ref)).Members {
		for _, c := range m.Comments {
			if c.ID == id {
				return c.Version
			}
		}
	}
	return ""
}

func (s standIn) add(ref models.ReviewRef, at models.Lines, pull int, body string) error {
	return s.queue.Add(ref, at, pull, body, s.build(ref))
}

func (s standIn) comments(ref models.ReviewRef, pull int) []models.Comment {
	for _, m := range s.review(ref).Members {
		if m.Pull == pull {
			return m.Comments
		}
	}
	return nil
}

func line(path, side string, n int) models.Lines {
	return models.Lines{Path: path, Side: side, Line: n}
}

func refusedLike(t *testing.T, err error, pattern string) {
	t.Helper()
	if err == nil || !regexp.MustCompile(pattern).MatchString(err.Error()) {
		t.Errorf("%v, not %s", err, pattern)
	}
}

func TestACommentIsQueuedOnALineOfTheDiffAndRefusedAnywhereElse(t *testing.T) {
	s := newStandIn(t)
	if err := s.add(single, line("docs/guide.md", "RIGHT", 3), 0, "  Every item, or every item shown?  "); err != nil {
		t.Fatal(err)
	}
	got := s.comments(single, 30)[1]
	if got != (models.Comment{ID: 2, Path: "docs/guide.md", Line: 3, Side: "RIGHT", Body: "Every item, or every item shown?", Code: "The catalog lists every item."}) {
		t.Errorf("the comment queued is %+v", got)
	}
	refusedLike(t, s.add(single, line("docs/guide.md", "RIGHT", 40), 0, "Hm"), `^not queued: GitHub takes comments only on lines of the diff, and this one is not`)
	refusedLike(t, s.add(single, line("docs/guide.md", "RIGHT", 3), 0, "   "), `^not queued: the text is empty$`)
	if err := s.add(single, line("docs/guide.md", "LEFT", 3), 0, "Why drop this?"); err != nil {
		t.Fatal(err)
	}
	if c := s.comments(single, 30); len(c) != 3 || c[2].ID != 3 || c[2].Side != "LEFT" {
		t.Errorf("the queue is %+v", c)
	}
}

func TestARangeMayStartOnARemovedLineAndPostsAsARange(t *testing.T) {
	s := newStandIn(t)
	refusedLike(t, s.add(single, models.Lines{Path: "docs/guide.md", Side: "RIGHT", Line: 4, StartLine: 5, StartSide: "RIGHT"}, 0, "Backwards?"), `^not queued: A range runs from its first line to its last, and this one does not\.$`)
	if err := s.add(single, models.Lines{Path: "docs/guide.md", Side: "RIGHT", Line: 3, StartLine: 3, StartSide: "LEFT"}, 0, "Why reword it?"); err != nil {
		t.Fatal(err)
	}
	if c := s.comments(single, 30)[1]; c.StartLine != 3 || c.StartSide != "LEFT" || c.Line != 3 || c.Side != "RIGHT" || c.Code != "The catalog lists items.\nThe catalog lists every item." {
		t.Fatalf("the range queued is %+v", c)
	}
	m := s.member(single, 30)
	if _, err := s.queue.Post(single, 30, "approve-with-comments", m.Sets["approve-with-comments"].Set, s.build(single)); err != nil {
		t.Fatal(err)
	}
	var body struct {
		Comments []map[string]any `json:"comments"`
	}
	_ = json.Unmarshal(s.gh.Posts()[0].Body, &body)
	if len(body.Comments) != 2 || body.Comments[1]["start_line"] != 3.0 || body.Comments[1]["start_side"] != "LEFT" {
		t.Errorf("the range posts as %v", body.Comments)
	}
}

func TestOnAStackACommentGoesToThePullRequestThatLastChangedItsLineAtThatPullRequestsOwnLine(t *testing.T) {
	s := newStandIn(t)
	if err := s.add(stack, line("Sources/Catalog/MemoryCatalog.swift", "RIGHT", 9), 0, "Should add check for duplicates?"); err != nil {
		t.Fatal(err)
	}
	first := s.comments(stack, 21)[1]
	if first.Shown == nil || *first.Shown != line("Sources/Catalog/MemoryCatalog.swift", "RIGHT", 9) {
		t.Errorf("the comment on #21 shows at %+v", first.Shown)
	}
	if err := s.add(stack, line("Sources/Basket/Basket.swift", "RIGHT", 7), 0, "Could this be passed in?"); err != nil {
		t.Fatal(err)
	}
	if c := s.comments(stack, 22); len(c) != 2 || c[1].Line != 7 {
		t.Errorf("#22's queue is %+v", c)
	}
	refusedLike(t, s.add(stack, line("Sources/Basket/Basket.swift", "RIGHT", 7), 21, "x"), `^not queued: #22 last changed this line, not #21`)
	if err := s.add(stack, models.Lines{Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 7, StartLine: 3, StartSide: "RIGHT"}, 0, "Both of these?"); err != nil {
		t.Fatal(err)
	}
	if c := s.comments(stack, 22)[2]; c.StartLine != 3 || c.Line != 7 || c.Code != "    let search: Search\n\n    init(catalog: Catalog) {\n        self.catalog = catalog\n        self.search = Search(catalog: catalog)" {
		t.Errorf("the range on #22 is %+v", c)
	}
	refusedLike(t, s.add(stack, line("Sources/Catalog/Catalog.swift", "RIGHT", 2), 0, "x"), `^not queued: This file is not in the change`)
	var ids [][]int
	for _, m := range s.review(stack).Members {
		var in []int
		for _, c := range m.Comments {
			in = append(in, c.ID)
		}
		ids = append(ids, in)
	}
	if len(ids) != 2 || len(ids[0]) != 2 || ids[0][1] != 3 || len(ids[1]) != 3 || ids[1][2] != 5 {
		t.Errorf("the comments are numbered %v", ids)
	}
}

func TestAChangeFromAWindowDrawnBeforeTheReviewWasLoadedAgainIsRefused(t *testing.T) {
	s := newStandIn(t)
	old := "0123456789ab"
	refused := `the review was loaded again after this window opened, so its lines may have moved; reload it, then try again`
	refusedLike(t, s.queue.Add(single, line("docs/guide.md", "RIGHT", 3), 0, "Round it.", old), `^not queued: `+refused)
	refusedLike(t, s.queue.Edit(single, 1, "Reworded.", s.version(single, 1), old), refused)
	refusedLike(t, s.queue.Drop(single, 1, s.version(single, 1), old), refused)
	_, err := s.queue.Post(single, 30, "approve", s.member(single, 30).Sets["approve"].Set, old)
	refusedLike(t, err, refused)
	refusedLike(t, s.queue.Add(single, line("docs/guide.md", "RIGHT", 3), 0, "Round it.", ""), `^not queued: the change does not name the page build its window was drawn from`)
	if c := s.comments(single, 30); len(c) != 1 || c[0].Body != "Every item, or every item shown?" {
		t.Errorf("something refused changed the queue: %+v", c)
	}
	where, _ := s.queue.Where(single, line("docs/guide.md", "RIGHT", 3), 0, old)
	if where.Refused != "The review was loaded again after this window opened, so its lines may have moved; reload it, then try again." {
		t.Errorf("where says %q", where.Refused)
	}
}

func TestACommentIsRefusedOnceItsPullRequestMovedOnGitHub(t *testing.T) {
	s := newStandIn(t)
	s.gh.Change(t, "GET repos/octo/widgets/pulls/30", func(pull map[string]any) { pull["head"].(map[string]any)["sha"] = strings.Repeat("f", 40) })
	refusedLike(t, s.add(single, line("docs/guide.md", "RIGHT", 3), 0, "Hm"), `^not queued: #30 moved from [0-9a-f]{8} to ffffffff; load it again`)
}

func TestAddCommentsPostsExactlyTheQueuedCommentsAsOneReviewAndOnlyWithTheSetThePageShowed(t *testing.T) {
	s := newStandIn(t)
	shown := s.member(single, 30).Sets
	if len(shown) != 3 {
		t.Fatalf("offered %v", shown)
	}
	_, err := s.queue.Post(single, 30, "comment", shown["approve"].Set, s.build(single))
	refusedLike(t, err, `^the page showed set [0-9a-f]{12}, but "comment" now gives set [0-9a-f]{12}; nothing was posted$`)
	if err := s.add(single, line("docs/guide.md", "LEFT", 3), 0, "And this one?"); err != nil {
		t.Fatal(err)
	}
	_, err = s.queue.Post(single, 30, "comment", shown["comment"].Set, s.build(single))
	refusedLike(t, err, `nothing was posted`)
	if len(s.gh.Posts()) != 0 {
		t.Fatalf("posted %v", s.gh.Posts())
	}
	s.gh.Route(t, "POST repos/octo/widgets/pulls/30/reviews", map[string]any{"id": 7, "state": "COMMENTED", "html_url": reviewURL})
	s.gh.Route(t, "GET repos/octo/widgets/pulls/30/comments?per_page=100&page=1", []map[string]any{{"pull_request_review_id": 7}, {"pull_request_review_id": 7}})
	done, err := s.queue.Post(single, 30, "comment", s.member(single, 30).Sets["comment"].Set, s.build(single))
	if err != nil || done.Message != "posted 2 comments as one review: "+reviewURL {
		t.Fatalf("posting says %q, %v", done.Message, err)
	}
	var body map[string]any
	_ = json.Unmarshal(s.gh.Posts()[0].Body, &body)
	text, _ := json.Marshal(body["comments"])
	if body["event"] != "COMMENT" || body["commit_id"] != s.review(single).Members[0].Head ||
		string(text) != `[{"body":"Every item, or every item shown?","line":3,"path":"docs/guide.md","side":"RIGHT"},{"body":"And this one?","line":3,"path":"docs/guide.md","side":"LEFT"}]` {
		t.Errorf("posted %v", body)
	}
	member := s.review(single).Members[0]
	if len(member.Comments) != 0 || len(member.Posted) != 1 || member.Posted[0].Review != reviewURL || len(member.Posted[0].Comments) != 2 || member.Posted[0].Action != "comment" {
		t.Errorf("the member after posting is %+v", member)
	}
	if sets := s.member(single, 30).Sets; len(sets) != 1 {
		t.Errorf("with nothing queued, %v is offered", sets)
	}
}

func TestApprovePostsAnApprovalAloneAndLeavesTheQueuedCommentsQueued(t *testing.T) {
	s := newStandIn(t)
	s.gh.Route(t, "POST repos/octo/widgets/pulls/30/reviews", map[string]any{"id": 70, "state": "APPROVED", "html_url": reviewURL})
	done, err := s.queue.Post(single, 30, "approve", s.member(single, 30).Sets["approve"].Set, s.build(single))
	if err != nil || done.Message != "approved, with no comments: "+reviewURL {
		t.Fatalf("approving says %q, %v", done.Message, err)
	}
	if body := string(s.gh.Posts()[0].Body); !strings.Contains(body, `"event":"APPROVE"`) || strings.Contains(body, "comments") {
		t.Errorf("posted %s", body)
	}
	if len(s.comments(single, 30)) != 1 {
		t.Errorf("the queued comment went")
	}
}

func TestNothingIsPostedToAClosedPullRequestAMovedHeadOrOnesOwnPullRequest(t *testing.T) {
	s := newStandIn(t)
	shown := s.member(single, 30).Sets
	s.gh.Change(t, "GET repos/octo/widgets/pulls/30", func(pull map[string]any) { pull["state"] = "closed" })
	_, err := s.queue.Post(single, 30, "comment", shown["comment"].Set, s.build(single))
	refusedLike(t, err, `^nothing was posted: #30 is closed$`)
	s.gh.Change(t, "GET repos/octo/widgets/pulls/30", func(pull map[string]any) {
		pull["state"] = "open"
		pull["head"].(map[string]any)["sha"] = strings.Repeat("f", 40)
	})
	_, err = s.queue.Post(single, 30, "comment", shown["comment"].Set, s.build(single))
	refusedLike(t, err, `the head moved from [0-9a-f]{8} to ffffffff\b`)
	if len(s.gh.Posts()) != 0 {
		t.Fatalf("posted %v", s.gh.Posts())
	}

	own := newStandIn(t)
	path := own.store.Home().Review(single).Record()
	text, _ := os.ReadFile(path)
	_ = os.WriteFile(path, []byte(strings.Replace(string(text), `"author": "alex"`, `"author": "reviewer"`, 1)), 0o644)
	if sets := own.member(single, 30).Sets; len(sets) != 1 || sets["comment"].Set == "" {
		t.Errorf("one's own pull request offers %v", sets)
	}
	_, err = own.queue.Post(single, 30, "approve", "abc", own.build(single))
	refusedLike(t, err, `^"approve" is not offered for #30; nothing was posted$`)
}

func TestAPressWhoseReviewGitHubAlreadyHoldsPostsNothingNewAndRecordsIt(t *testing.T) {
	s := newStandIn(t)
	head := s.review(single).Members[0].Head
	s.gh.Route(t, "GET repos/octo/widgets/pulls/30/reviews?per_page=100&page=1", []map[string]any{{"id": 9, "user": map[string]any{"login": "reviewer"}, "commit_id": head, "state": "COMMENTED", "html_url": reviewURL}})
	s.gh.Route(t, "GET repos/octo/widgets/pulls/30/comments?per_page=100&page=1", []map[string]any{{"pull_request_review_id": 9, "path": "docs/guide.md", "body": "Every item, or every item shown?\n"}})
	done, err := s.queue.Post(single, 30, "comment", s.member(single, 30).Sets["comment"].Set, s.build(single))
	if err != nil || done.Message != "already done as "+reviewURL+"; posted nothing new" {
		t.Fatalf("pressing again says %q, %v", done.Message, err)
	}
	member := s.review(single).Members[0]
	if len(s.gh.Posts()) != 0 || len(member.Comments) != 0 || len(member.Posted) != 1 || member.Posted[0].Review != reviewURL {
		t.Errorf("the member is %+v, posted %v", member, s.gh.Posts())
	}
}

func TestNothingIsPostedWhenGitHubAnswersAsAnotherAccount(t *testing.T) {
	s := newStandIn(t)
	s.gh.Route(t, "GET user", map[string]any{"login": "someone-else"})
	_, err := s.queue.Post(single, 30, "approve", s.member(single, 30).Sets["approve"].Set, s.build(single))
	refusedLike(t, err, `^GitHub answers as someone-else now, but this review was loaded as reviewer; nothing was posted$`)
}

func TestOnAStackEachPullRequestPostsItsOwnReviewWithItsOwnSet(t *testing.T) {
	s := newStandIn(t)
	s21, s22 := s.member(stack, 21).Sets, s.member(stack, 22).Sets
	if len(s21) != 3 || len(s22) != 3 {
		t.Fatalf("offered %v and %v", s21, s22)
	}
	_, err := s.queue.Post(stack, 21, "comment", s22["comment"].Set, s.build(stack))
	refusedLike(t, err, `^the page showed set`)
	s.gh.Route(t, "POST repos/octo/widgets/pulls/22/reviews", map[string]any{"id": 11, "state": "APPROVED", "html_url": "https://github.com/octo/widgets/pull/22#r"})
	s.gh.Route(t, "GET repos/octo/widgets/pulls/22/comments?per_page=100&page=1", []map[string]any{{"pull_request_review_id": 11}})
	done, err := s.queue.Post(stack, 22, "approve-with-comments", s22["approve-with-comments"].Set, s.build(stack))
	if err != nil || !strings.HasPrefix(done.Message, "approved with 1 comment: ") {
		t.Fatalf("approving #22 says %q, %v", done.Message, err)
	}
	if posts := s.gh.Posts(); len(posts) != 1 || posts[0].Path != "repos/octo/widgets/pulls/22/reviews" || !strings.Contains(string(posts[0].Body), `"start_line":9`) {
		t.Errorf("posted %v", posts)
	}
}

func TestAnEditOrADropNamesTheCommentsVersionAndIsRefusedOnceAnotherWindowChangedIt(t *testing.T) {
	s := newStandIn(t)
	build := s.build(single)
	seen := s.version(single, 1)
	if err := s.queue.Edit(single, 1, "Changed in the other window.", seen, build); err != nil {
		t.Fatal(err)
	}
	refusedLike(t, s.queue.Edit(single, 1, "Mine.", seen, build), `^not changed: comment 1 was changed after this window showed it, and now reads "Changed in the other window\."$`)
	refusedLike(t, s.queue.Drop(single, 1, seen, build), `^not dropped: comment 1 was changed after this window showed it`)
	refusedLike(t, s.queue.Edit(single, 1, "Mine.", "", build), `^not changed: the change does not name the version of comment 1 it was made on$`)
	if c := s.comments(single, 30); c[0].Body != "Changed in the other window." {
		t.Errorf("something refused changed it: %q", c[0].Body)
	}
	if err := s.queue.Edit(single, 1, " Mine. ", s.version(single, 1), build); err != nil {
		t.Fatal(err)
	}
	refusedLike(t, s.queue.Edit(single, 1, " ", s.version(single, 1), build), `^not changed: the text is empty$`)
	refusedLike(t, s.queue.Drop(single, 9, "abc", build), `^no queued comment 9$`)
	if err := s.queue.Drop(single, 1, s.version(single, 1), build); err != nil {
		t.Fatal(err)
	}
	if c := s.comments(single, 30); len(c) != 0 {
		t.Errorf("the queue is %+v", c)
	}
	if err := s.add(single, line("docs/guide.md", "RIGHT", 3), 0, "Again."); err != nil || s.comments(single, 30)[0].ID != 1 {
		t.Errorf("a new comment takes the number after the highest still held: %+v, %v", s.comments(single, 30), err)
	}
}
