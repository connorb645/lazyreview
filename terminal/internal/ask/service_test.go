package ask

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/commands/claude"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/testsupport"
)

const guide = "# Guide\n\nThe catalog lists every item.\n\n- Browse\n- Search\n"

func TestMain(m *testing.M) {
	testsupport.ActAsFakeClaude()
	os.Exit(m.Run())
}

// world is a data home holding reviews of a throwaway repository, with
// lazyreview's own clone of it, and the service that asks the stand-in
// Claude about them.
type world struct {
	t       *testing.T
	repo    *testsupport.ThrowawayRepo
	home    store.DataHome
	reviews *store.ReviewStore
	fake    *testsupport.FakeClaude
	service *Service
}

func newWorld(t *testing.T) *world {
	t.Helper()
	tr := i18n.EnglishTranslationSet()
	repo := testsupport.NewThrowawayRepo(t)
	repo.Commit("main", map[string]string{"docs/guide.md": "# Guide\n\nThe catalog lists items.\n\n- Browse\n- Search\n", "Sources/Catalog/Catalog.swift": "protocol Catalog {}\n",
		"Sources/Basket/Basket.swift": "final class Basket {\n}\n"}, "base")
	repo.Commit("fix/wording", map[string]string{"docs/guide.md": guide}, "Fix the guide wording")
	repo.Checkout("main")
	repo.Commit("feature/search", map[string]string{"Sources/Catalog/Search.swift": "struct Search {\n  let catalog: Catalog\n\n  func find() {}\n}\n"}, "Add search")
	repo.Commit("feature/basket", map[string]string{"Sources/Basket/Basket.swift": "final class Basket {\n  let search: Search\n}\n"}, "Add a basket that uses search")
	repo.Checkout("main")
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	home := store.DataHome{Root: root}
	clone := home.Clone("github.com", "octo/widgets")
	if out, err := exec.Command("git", "clone", "--quiet", "--bare", repo.Work, clone).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	locks := store.NewLocks(tr)
	reviews := store.NewReviewStore(tr, home, locks)
	fake := testsupport.NewFakeClaude(t)
	w := &world{t: t, repo: repo, home: home, reviews: reviews, fake: fake}
	w.service = NewService(tr, reviews, locks, claude.New(tr, fake.Command, ""), nil, nil, nil, func() string { return "2026-10-15T13:00:00.000Z" })
	return w
}

func (w *world) write(ref models.ReviewRef, rev models.Review, page any) {
	w.t.Helper()
	folder := w.home.Review(ref)
	for path, v := range map[string]any{folder.Record(): rev, folder.Page(): page} {
		text, err := json.Marshal(v)
		if err == nil {
			err = os.MkdirAll(filepath.Dir(path), 0o755)
		}
		if err == nil {
			err = os.WriteFile(path, text, 0o644)
		}
		if err != nil {
			w.t.Fatal(err)
		}
	}
}

func (w *world) single() (models.ReviewRef, string) {
	w.t.Helper()
	ref, _ := models.ParseReviewRef(i18n.EnglishTranslationSet(), "github.com/octo/widgets/30")
	main, head := w.repo.Git("rev-parse", "main"), w.repo.Git("rev-parse", "fix/wording")
	rev := models.Review{ID: ref.ID(), Host: "github.com", Repo: "octo/widgets", Me: "reviewer", Title: "Fix the guide wording", Base: models.Ref{Ref: "main", Sha: main},
		Members: []models.ReviewMember{{Pull: 30, Head: head, Start: main, Title: "Fix the guide wording", Author: "alex", Base: "main", State: "open", Color: 1}}}
	page := models.PageRecord{Model: models.Model{Files: []models.FileEntry{{Path: "docs/guide.md", OldPath: "docs/guide.md", Status: "M"}}},
		Diff:    map[string]string{"docs/guide.md": "diff --git a/docs/guide.md b/docs/guide.md\n@@ -1,6 +1,6 @@\n # Guide\n \n-The catalog lists items.\n+The catalog lists every item.\n \n - Browse\n - Search\n"},
		Sources: map[string][]string{"docs/guide.md": strings.Split(strings.TrimSuffix(guide, "\n"), "\n")}, Context: map[string][]string{}}
	w.write(ref, rev, page)
	build, err := w.reviews.PageBuild(ref)
	if err != nil {
		w.t.Fatal(err)
	}
	return ref, build
}

func (w *world) stack() (models.ReviewRef, string) {
	w.t.Helper()
	ref, _ := models.ParseReviewRef(i18n.EnglishTranslationSet(), "github.com/octo/widgets/21")
	main, low, top := w.repo.Git("rev-parse", "main"), w.repo.Git("rev-parse", "feature/search"), w.repo.Git("rev-parse", "feature/basket")
	rev := models.Review{ID: ref.ID(), Host: "github.com", Repo: "octo/widgets", Me: "reviewer", Title: "Search, then a basket", Base: models.Ref{Ref: "main", Sha: main},
		Members: []models.ReviewMember{{Pull: 21, Head: low, Start: main, Title: "Add search", State: "open", Color: 1}, {Pull: 22, Head: top, Start: low, Title: "Add a basket that uses search", State: "open", Color: 2}}}
	page := map[string]any{
		"model": models.Model{Files: []models.FileEntry{{Path: "Sources/Catalog/Search.swift", Status: "A"}, {Path: "Sources/Basket/Basket.swift", OldPath: "Sources/Basket/Basket.swift", Status: "M"}}},
		"diff":  map[string]string{}, "sources": map[string][]string{}, "context": map[string][]string{},
		"stack": map[string]any{"members": []any{}, "warnings": []any{}, "lines": map[string]any{
			"Sources/Catalog/Search.swift": map[string]any{"RIGHT": map[string]any{"4": []any{21, 4}}},
			"Sources/Basket/Basket.swift":  map[string]any{"RIGHT": map[string]any{"2": []any{22, 2}}},
		}},
	}
	w.write(ref, rev, page)
	build, _ := w.reviews.PageBuild(ref)
	return ref, build
}

func (w *world) branch() (models.ReviewRef, string) {
	w.t.Helper()
	ref, _ := models.ParseReviewRef(i18n.EnglishTranslationSet(), "local/work-1a2b3c4d/fix~wording")
	main, head := w.repo.Git("rev-parse", "main"), w.repo.Git("rev-parse", "fix/wording")
	rev := models.Review{ID: ref.ID(), Kind: "branch", Repo: "work", Path: w.repo.Work, Title: "fix/wording", Base: models.Ref{Ref: "main", Sha: main}, Head: models.Ref{Ref: "fix/wording", Sha: head}, Start: main}
	page := models.PageRecord{Model: models.Model{Files: []models.FileEntry{{Path: "docs/guide.md", OldPath: "docs/guide.md", Status: "M"}}}, Diff: map[string]string{}, Sources: map[string][]string{}, Context: map[string][]string{}}
	w.write(ref, rev, page)
	build, _ := w.reviews.PageBuild(ref)
	return ref, build
}

func (w *world) threads(ref models.ReviewRef) []models.Conversation {
	w.t.Helper()
	threads, err := w.reviews.Threads(ref)
	if err != nil {
		w.t.Fatal(err)
	}
	return threads
}

func line(path string, n int) models.Lines {
	return models.Lines{Path: path, Side: "RIGHT", Line: n}
}

func said(c models.Conversation) [][2]string {
	var out [][2]string
	for _, m := range c.Messages {
		out = append(out, [2]string{m.From, m.Text + m.Error})
	}
	return out
}

func TestAConversationKeepsTheQuestionThenClaudesAnswerOrWhyThereIsNone(t *testing.T) {
	w := newWorld(t)
	ref, build := w.single()
	c, err := w.service.Start(ref, line("docs/guide.md", 3), " Every? ", build)
	if err != nil || !reflect.DeepEqual(said(c), [][2]string{{"you", "Every?"}, {"claude", "It returns early when the list is empty."}}) {
		t.Fatalf("the conversation is %v, %v", said(c), err)
	}
	matches(t, w.fake.Asked()[0].Prompt, `Pull request #30: Fix the guide wording[\s\S]*File: docs/guide\.md \(changed\)`, "the question")
	w.fake.Fail()
	again, err := w.service.Reply(ref, c.ID, "And then?", build)
	if err != nil || !strings.Contains(again.Messages[3].Error, "could not answer") || len(w.threads(ref)) != 1 {
		t.Errorf("a failed answer reads %v, %v", said(again), err)
	}
	if _, err := w.service.Start(ref, line("docs/guide.md", 3), "  ", build); err == nil || err.Error() != "ask something first" {
		t.Errorf("an empty question gave %v", err)
	}
	if _, err := w.service.Reply(ref, 9, "Hello?", build); err == nil || !strings.Contains(err.Error(), "there is no conversation 9") {
		t.Errorf("a missing conversation gave %v", err)
	}
	if _, err := w.service.Start(ref, line("docs/guide.md", 3), "Why?", "0123456789ab"); err == nil || !strings.HasPrefix(err.Error(), "not asked: the review was loaded again after this window opened") {
		t.Errorf("an old window gave %v", err)
	}
	if _, err := w.service.Reply(ref, c.ID, "Still?", "0123456789ab"); err == nil || len(w.threads(ref)[0].Messages) != 4 {
		t.Errorf("a refused follow-up was kept, or gave %v", err)
	}
}

func TestClaudeWorksInACopyAtThePullRequestsHeadOnePerConversationAndAFollowUpResumes(t *testing.T) {
	w := newWorld(t)
	ref, build := w.single()
	c, err := w.service.Start(ref, line("docs/guide.md", 3), "Where is the catalog?", build)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.service.Reply(ref, c.ID, "And its test?", build); err != nil {
		t.Fatal(err)
	}
	other, err := w.service.Start(ref, line("docs/guide.md", 5), "Why a list?", build)
	if err != nil {
		t.Fatal(err)
	}
	calls := w.fake.Asked()
	first, followUp, second := calls[0], calls[1], calls[2]
	if first.Folder != w.home.Copy(ref, 1) || first.Files["docs/guide.md"] != guide || !strings.Contains(first.Files["Sources/Catalog/Catalog.swift"], "protocol Catalog") {
		t.Errorf("Claude worked in %s on %v", first.Folder, first.Files)
	}
	branches := "main " + w.repo.Git("rev-parse", "main") + "\npull/30 " + w.repo.Git("rev-parse", "fix/wording")
	if !reflect.DeepEqual(first.Git, testsupport.FakeGit{Branch: "pull/30", Branches: branches}) {
		t.Errorf("the copy's git says %+v", first.Git)
	}
	if !strings.Contains(first.Prompt, "The codebase as it stands after this change is in "+w.home.Copy(ref, 1)+", your working folder, with the branch pull/30 checked out") {
		t.Errorf("the question says %q", first.Prompt)
	}
	var settings struct {
		Hooks struct {
			PreToolUse []struct {
				Hooks []struct {
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"PreToolUse"`
		} `json:"hooks"`
	}
	_ = json.Unmarshal([]byte(at(first.Args, "--settings")), &settings)
	if hook := settings.Hooks.PreToolUse[0].Hooks[0].Command; !strings.HasSuffix(hook, " guard "+quoted(w.home.Copy(ref, 1))) {
		t.Errorf("the guard is %q", hook)
	}
	session := w.threads(ref)[0].Session
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(session) || at(first.Args, "--session-id") != session {
		t.Errorf("the session is %q, asked as %q", session, at(first.Args, "--session-id"))
	}
	if followUp.Folder != first.Folder || at(followUp.Args, "--resume") != session || followUp.Prompt != "Their follow-up question, on the same lines:\nAnd its test?" {
		t.Errorf("the follow-up was %+v", followUp)
	}
	if second.Folder != w.home.Copy(ref, 2) || other.Session == session {
		t.Errorf("a second conversation shares %s or its session", second.Folder)
	}
}

func TestOnAStackTheCopyIsAtTheHeadOfThePullRequestThatChangedTheLines(t *testing.T) {
	w := newWorld(t)
	ref, build := w.stack()
	if c, err := w.service.Start(ref, line("Sources/Catalog/Search.swift", 4), "Who calls find?", build); err != nil || c.Messages[1].Error != "" {
		t.Fatalf("the lower question gave %v, %v", said(c), err)
	}
	if _, err := w.service.Start(ref, line("Sources/Basket/Basket.swift", 2), "Why keep a search?", build); err != nil {
		t.Fatal(err)
	}
	calls := w.fake.Asked()
	lower, upper := calls[0], calls[1]
	if lower.Git.Branch != "pull/21" || strings.Contains(lower.Files["Sources/Basket/Basket.swift"], "search") || !strings.Contains(lower.Prompt, "The codebase as it stands after #21 is in ") {
		t.Errorf("the lower question was asked on %s with %q", lower.Git.Branch, lower.Files["Sources/Basket/Basket.swift"])
	}
	if upper.Git.Branch != "pull/22" || !strings.Contains(upper.Files["Sources/Basket/Basket.swift"], "let search: Search") {
		t.Errorf("the upper question was asked on %s", upper.Git.Branch)
	}
}

func TestOnABranchReviewTheCopyIsAtTheBranchsCommitFromLazyreviewsOwnClone(t *testing.T) {
	w := newWorld(t)
	w.repo.Checkout("fix/wording")
	ref, build := w.branch()
	if err := os.WriteFile(filepath.Join(w.repo.Work, "docs/guide.md"), []byte("Not committed.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.repo.Work, "notes.txt"), []byte("Not added.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := testsupport.FolderState(t, w.repo.Work)
	c, err := w.service.Start(ref, line("docs/guide.md", 3), "Every?", build)
	if err != nil || c.Messages[1].Text != "It returns early when the list is empty." {
		t.Fatalf("the conversation is %v, %v", said(c), err)
	}
	asked := w.fake.Asked()[0]
	if asked.Folder != w.home.Copy(ref, 1) || asked.Files["docs/guide.md"] != guide || asked.Files["notes.txt"] != "" || asked.Git.Branch != "fix/wording" || asked.Git.Remotes != "" {
		t.Errorf("Claude worked in %s on %v, %+v", asked.Folder, asked.Files, asked.Git)
	}
	alternates, _ := os.ReadFile(filepath.Join(asked.Folder, ".git/objects/info/alternates"))
	if !strings.Contains(string(alternates), "clones/local/work-1a2b3c4d.git/objects") {
		t.Errorf("the copy borrows from %q", alternates)
	}
	if !reflect.DeepEqual(testsupport.FolderState(t, w.repo.Work), before) {
		t.Error("the worktree or its .git folder changed")
	}
}

func TestTheAnswerKeepsTheStepsClaudeTookEachRefusedOneWithWhy(t *testing.T) {
	w := newWorld(t)
	ref, build := w.single()
	w.fake.Answer(testsupport.FakeAnswer{Text: "Only the guide says so.", Steps: []testsupport.FakeStep{
		{Tool: "Grep", Input: map[string]any{"pattern": "catalog", "path": w.home.Copy(ref, 1) + "/docs"}},
		{Tool: "Bash", Input: map[string]any{"command": `gh pr comment 30 --body "Every item?"`}, Refused: "lazyreview refused this: gh may only read, one command at a time."},
	}})
	c, err := w.service.Start(ref, line("docs/guide.md", 3), "Every?", build)
	want := []models.Step{{Words: "Searched for catalog in docs"}, {Words: `Ran gh pr comment 30 --body "Every item?"`, Refused: "gh may only read, one command at a time."}}
	if err != nil || !reflect.DeepEqual(c.Messages[1].Steps, want) || c.Working != nil {
		t.Errorf("the answer kept %+v, working %+v, %v", c.Messages[1].Steps, c.Working, err)
	}
}

func TestWhileClaudeWorksTheConversationShowsItsStepsAndAStopSaysSo(t *testing.T) {
	w := newWorld(t)
	ref, build := w.single()
	w.fake.Answer(testsupport.FakeAnswer{Text: "Too late.", Steps: []testsupport.FakeStep{{Tool: "Bash", Input: map[string]any{"command": "npm test"}}}, Wait: 60_000})
	asking := make(chan models.Conversation)
	go func() {
		c, _ := w.service.Start(ref, line("docs/guide.md", 3), "Is it tested?", build)
		asking <- c
	}()
	var working *models.Working
	for i := 0; i < 100 && (working == nil || len(working.Steps) == 0); i++ {
		time.Sleep(100 * time.Millisecond)
		if threads := w.threads(ref); len(threads) > 0 {
			working = threads[0].Working
		}
	}
	if working == nil || !reflect.DeepEqual(working.Steps, []models.Step{{Words: "Ran npm test"}}) {
		t.Fatalf("while Claude works the conversation shows %+v", working)
	}
	if _, err := w.service.Reply(ref, 1, "And?", build); err == nil || !strings.HasPrefix(err.Error(), "Claude is still working on this conversation") {
		t.Errorf("a follow-up while Claude works gave %v", err)
	}
	stopped, err := w.service.Stop(ref, 1)
	if err != nil || !reflect.DeepEqual(said(stopped), [][2]string{{"you", "Is it tested?"}, {"claude", "you stopped it"}}) || stopped.Working != nil ||
		!reflect.DeepEqual(stopped.Messages[1].Steps, []models.Step{{Words: "Ran npm test"}}) {
		t.Errorf("the stop gave %v, %+v, %v", said(stopped), stopped.Messages, err)
	}
	if c := <-asking; len(c.Messages) != 2 {
		t.Errorf("the question returned %v", said(c))
	}
	left := w.threads(ref)[0]
	left.ID, left.Messages, left.Working = 2, []models.Message{{From: "you", Text: "Asked before a restart", At: "x"}}, &models.Working{Steps: []models.Step{}}
	write, _ := w.reviews.ThreadWrite(ref, left)
	if err := os.WriteFile(write.Path, []byte(write.Text), 0o644); err != nil {
		t.Fatal(err)
	}
	unstuck, err := w.service.Stop(ref, 2)
	if err != nil || !reflect.DeepEqual(said(unstuck), [][2]string{{"you", "Asked before a restart"}, {"claude", "you stopped it"}}) || unstuck.Working != nil {
		t.Errorf("a conversation left waiting stopped as %v, %+v, %v", said(unstuck), unstuck.Working, err)
	}
}

func TestAFollowUpWhoseSessionIsGoneStartsANewOneWithTheWholeQuestion(t *testing.T) {
	w := newWorld(t)
	ref, build := w.single()
	if _, err := w.service.Start(ref, line("docs/guide.md", 3), "Every?", build); err != nil {
		t.Fatal(err)
	}
	lost := w.threads(ref)[0]
	lost.Session = "00000000-0000-4000-8000-000000000000"
	write, _ := w.reviews.ThreadWrite(ref, lost)
	if err := os.WriteFile(write.Path, []byte(write.Text), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := w.service.Reply(ref, 1, "And then?", build)
	if err != nil {
		t.Fatal(err)
	}
	calls := w.fake.Asked()
	gone, fresh := calls[1], calls[2]
	if at(gone.Args, "--resume") != lost.Session || at(fresh.Args, "--session-id") != c.Session || c.Session == lost.Session {
		t.Errorf("the lost session was %q, the new one %q", at(gone.Args, "--resume"), at(fresh.Args, "--session-id"))
	}
	matches(t, fresh.Prompt, `The conversation so far:\nThey asked: Every\?[\s\S]*Their question:\nAnd then\?$`, "the whole question")
	if c.Messages[3].Text != "It returns early when the list is empty." {
		t.Errorf("the answer is %v", said(c))
	}
}

func TestWhenNoCopyOfTheCodeCanBeMadeTheConversationSaysWhyAndClaudeIsNotAsked(t *testing.T) {
	w := newWorld(t)
	ref, build := w.branch()
	if err := os.Rename(w.repo.Work, w.repo.Work+"-moved"); err != nil {
		t.Fatal(err)
	}
	c, err := w.service.Start(ref, line("docs/guide.md", 3), "Every?", build)
	if err := os.Rename(w.repo.Work+"-moved", w.repo.Work); err != nil {
		t.Fatal(err)
	}
	if err != nil || !regexp.MustCompile(`^could not make a copy of the code for Claude: git fetch failed: .*does not appear to be a git repository`).MatchString(c.Messages[1].Error) {
		t.Errorf("the conversation says %v, %v", said(c), err)
	}
	if len(w.fake.Asked()) != 0 {
		t.Error("Claude was asked")
	}
}

func TestAnAnswerThatEndsWithoutOneSaysSo(t *testing.T) {
	w := newWorld(t)
	ref, build := w.single()
	w.service.answerer = func(models.ReviewRef, int) error { return errors.New("signal: killed") }
	c, err := w.service.Start(ref, line("docs/guide.md", 3), "Every?", build)
	if err != nil || !reflect.DeepEqual(said(c), [][2]string{{"you", "Every?"}, {"claude", "the answer stopped before Claude gave one: signal: killed"}}) {
		t.Errorf("the conversation is %v, %v", said(c), err)
	}
}

func at(args []string, name string) string {
	for i, a := range args {
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
