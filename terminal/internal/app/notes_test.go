package app

import (
	"os"
	"strings"
	"testing"

	"review-desk/terminal/internal/server"
)

func onBasket(t *testing.T, set ...func(*Options)) App {
	t.Helper()
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30, set...)
	a = typed(t, press(t, a, "space", "f"), "Basket/Basket")
	if a.screen != screenDiff || cursorAt(a) != "Sources/Basket/Basket.swift file" {
		t.Fatalf("the finder opened %v at %s", a.screen, cursorAt(a))
	}
	return a
}

func noteKeyAt(a App) string {
	r := a.diff.rows[a.diff.cursor]
	return r.kind + " " + r.key
}

func TestNotesShowUnderTheirLines(t *testing.T) {
	a := onBasket(t)
	screenIs(t, a, "notes")
	a = press(t, a, "]", "n")
	if got := noteKeyAt(a); got != "notehead conversation:1" {
		t.Fatalf("]n went to %s", got)
	}
	a = press(t, a, "z", "a", "]", "n", "z", "a")
	screenIs(t, a, "notes-open")
	a = press(t, a, "[", "n", "z", "a")
	if a.diff.noteOpen(note{key: "conversation:1", kind: "conversation"}) {
		t.Fatal("za folds the conversation again")
	}
}

func TestACommentIsWrittenInTheEditorAndQueued(t *testing.T) {
	editor, seen := writer(t, "Name the search for what it finds.")
	a := onBasket(t, editor)
	a = typed(t, press(t, a, ":"), "3")
	a = press(t, a, "c")
	file, err := os.ReadFile(seen)
	if err != nil {
		t.Fatalf("the editor was not opened: %v", err)
	}
	want := "\n\n" + cutLine + "\n# Nothing below the line above is sent. Saving with no text sends nothing.\n# A comment on Sources/Basket/Basket.swift, line 3.\n# Goes on #22, line 3.\n#\n#      3 +     let search: Search\n"
	if string(file) != want {
		t.Fatalf("the editor was given:\n%s\nnot:\n%s", file, want)
	}
	if a.notice != "Queued: A comment on Sources/Basket/Basket.swift, line 3" || a.bad {
		t.Fatalf("the status line says %q", a.notice)
	}
	screenIs(t, a, "comment-queued")
}

func TestASelectionIsCommentedOnAsARange(t *testing.T) {
	editor, _ := writer(t, "One add is enough.")
	a := onBasket(t, editor)
	a = press(t, a, "]", "f")
	a = typed(t, press(t, a, ":"), "7")
	a = press(t, a, "V", "j", "j")
	screenIs(t, a, "visual")
	a = press(t, a, "o")
	if at, _, _ := a.selected(); at.StartLine != 7 || at.Line != 9 {
		t.Fatalf("o keeps the same lines: %+v", at)
	}
	a = press(t, a, "c")
	if a.bad || a.diff.selecting {
		t.Fatalf("the range was not queued: %q", a.notice)
	}
	if n, ok := noteOf(*a.review, "Sources/Catalog/MemoryCatalog.swift", "queued:4"); !ok || n.at.StartLine != 7 || n.at.Line != 9 {
		t.Fatalf("queued comment 4 shows on %+v", n.at)
	}
}

func TestACommentWhereNoneCanGoSaysWhyFirst(t *testing.T) {
	editor, seen := writer(t, "Never sent.")
	a := onBasket(t, editor)
	a = typed(t, press(t, a, "space", "f"), "Catalog/Catalog.swift")
	a = typed(t, press(t, a, ":"), "1")
	a = press(t, a, "c")
	if !a.bad || a.notice != "This file is not in the change, so a comment cannot go on it. Ask Claude about it instead." {
		t.Fatalf("the status line says %q", a.notice)
	}
	if _, err := os.Stat(seen); err == nil {
		t.Fatal("the editor opened for a comment that cannot go anywhere")
	}
}

func TestSavingNothingSendsNothing(t *testing.T) {
	editor, _ := writer(t, "")
	a := onBasket(t, editor)
	a = press(t, typed(t, press(t, a, ":"), "3"), "c")
	if a.notice != "Nothing was written, so nothing was sent." {
		t.Fatalf("the status line says %q", a.notice)
	}
}

func TestARefusedEditKeepsItsTextAndSavesOverTheNewVersion(t *testing.T) {
	editor, _ := writer(t, "Keep the order the items came in.")
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30, editor)
	state := a.review.state
	queue := *state.Queue
	queue.Members = append([]server.QueueMember{}, queue.Members...)
	queue.Members[0].Comments = append([]server.Comment{}, queue.Members[0].Comments...)
	queue.Members[0].Comments[0].Version = "stale1"
	queue.Members[0].Comments[0].Body = ""
	state.Queue = &queue
	a = a.withState(state)
	a = press(t, a, "]", "]")
	for a.rv.view.items[a.rv.view.cursor].kind != "comment" {
		a = press(t, a, "j")
	}
	a = press(t, a, "e")
	if !a.bad || !strings.HasPrefix(a.notice, "not changed: comment 1 was changed after this window showed it") {
		t.Fatalf("the status line says %q", a.notice)
	}
	if a.drafts["edit 1"] != "Keep the order the items came in." {
		t.Fatalf("the text was not kept: %q", a.drafts)
	}
	keep := Options{}
	keep.Editor = "true"
	a.opts.Editor = keep.Editor
	a = press(t, a, "e")
	if a.bad || a.notice != "Saved queued comment 1." {
		t.Fatalf("saving again over the new version: %q", a.notice)
	}
	if _, ok := a.drafts["edit 1"]; ok {
		t.Fatal("the draft stays after it was saved")
	}
}

func TestDropAsksFirst(t *testing.T) {
	a := onBasket(t)
	a = press(t, a, "]", "n", "]", "n", "]", "n")
	if got := noteKeyAt(a); got != "notehead queued:2" {
		t.Fatalf("on %s", got)
	}
	a = press(t, a, "d", "d")
	screenIs(t, a, "drop-asks")
	a = press(t, a, "n")
	if a.notice != "Kept queued comment 2." {
		t.Fatalf("the status line says %q", a.notice)
	}
	a = press(t, a, "d", "d", "y")
	if a.notice != "Dropped queued comment 2." {
		t.Fatalf("the status line says %q", a.notice)
	}
	if _, ok := noteOf(*a.review, "Sources/Basket/Basket.swift", "queued:2"); ok {
		t.Fatal("the dropped comment still shows")
	}
}

func TestClaudeIsAskedAndAFollowUpGoesInTheConversation(t *testing.T) {
	editor, seen := writer(t, "Why keep a search here?")
	a := onBasket(t, editor)
	a = press(t, typed(t, press(t, a, ":"), "3"), "a")
	file, _ := os.ReadFile(seen)
	if !strings.Contains(string(file), "# A question to Claude about Sources/Basket/Basket.swift, line 3.\n# Claude answers here, in the review. Nothing goes to GitHub.\n") {
		t.Fatalf("the editor was given:\n%s", file)
	}
	if a.notice != "Claude answered." {
		t.Fatalf("the status line says %q", a.notice)
	}
	screenIs(t, a, "asked")
	follow, seen := writer(t, "Could it be passed in?")
	follow(&a.opts)
	a = press(t, a, "]", "n", "]", "n")
	if got := noteKeyAt(a); got != "notehead conversation:1" {
		t.Fatalf("on %s", got)
	}
	a = press(t, a, "a")
	file, _ = os.ReadFile(seen)
	if !strings.Contains(string(file), "# A follow-up to Claude on Sources/Basket/Basket.swift, line 7.\n# The conversation so far:\n#\n#   You: Could this be passed in?\n") {
		t.Fatalf("the editor was given:\n%s", file)
	}
	if n, _ := noteOf(*a.review, "Sources/Basket/Basket.swift", "conversation:1"); len(n.conversation.Messages) != 4 {
		t.Fatalf("the follow-up and its answer: %+v", n.conversation.Messages)
	}
}

func TestClaudesAnswerBecomesTheComment(t *testing.T) {
	a := onBasket(t, func(o *Options) { o.Editor = "true" })
	a = press(t, a, "]", "n", "z", "a", "j", "j")
	if r := a.diff.rows[a.diff.cursor]; r.part != "claude" {
		t.Fatalf("on %+v", r)
	}
	a = press(t, a, "c")
	if a.bad || a.notice != "Queued: A comment on Sources/Basket/Basket.swift, line 7, from Claude's answer" {
		t.Fatalf("the status line says %q", a.notice)
	}
}

func TestALineSeveralPullRequestsShowAsksWhichFirst(t *testing.T) {
	a := onBasket(t)
	e := editing{purpose: "comment", at: server.Lines{Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 2}, key: "comment", about: []string{"A comment on Sources/Basket/Basket.swift, line 2."}}
	a = send(t, a, checked{e: e, where: server.Where{Words: "Choose the pull request it goes on.", Choose: []server.Choice{{Pull: 21, Path: "Sources/Basket/Basket.swift", Line: 2}, {Pull: 22, Path: "Sources/Basket/Basket.swift", Line: 2}}}})
	if a.panel == nil || len(a.panel.rows) != 2 || a.panel.title != "Choose the pull request it goes on." {
		t.Fatalf("the choice: %+v", a.panel)
	}
	a = press(t, a, "j", "enter")
	if !strings.Contains(a.notice, "pull=22") {
		t.Fatalf("the choice was not asked about again with its pull request: %q", a.notice)
	}
}
