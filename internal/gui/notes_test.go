package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/models"
)

func onBasket(t *testing.T, set ...func(*Options)) App {
	t.Helper()
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30, set...)
	a = typed(t, press(t, a, "space", "f"), "Basket/Basket")
	if a.Screen() != context.ScreenDiff || cursorAt(a) != "Sources/Basket/Basket.swift file" {
		t.Fatalf("the finder opened %v at %s", a.Screen(), cursorAt(a))
	}
	return a
}

func noteKeyAt(a App) string {
	r := a.Diff.Rows[a.Diff.Cursor]
	return r.Kind + " " + r.Key
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
	if a.Diff.NoteOpen(presentation.Note{Key: "conversation:1", Kind: "conversation"}) {
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
	want := "\n\n# ------------------------ >8 ------------------------\n# Nothing below the line above is sent. Saving with no text sends nothing.\n# A comment on Sources/Basket/Basket.swift, line 3.\n# Goes on #22, line 3.\n#\n#      3 +     let search: Search\n"
	if string(file) != want {
		t.Fatalf("the editor was given:\n%s\nnot:\n%s", file, want)
	}
	if a.Notice != "Queued: A comment on Sources/Basket/Basket.swift, line 3" || a.Bad {
		t.Fatalf("the status line says %q", a.Notice)
	}
	screenIs(t, a, "comment-queued")
}

func TestASelectionIsCommentedOnAsARange(t *testing.T) {
	editor, _ := writer(t, "One add is enough.")
	a, _ := opened(t, stack, 100, 30, editor)
	a = typed(t, press(t, a, "space", "f"), "Catalog/MemoryCatalog")
	a = typed(t, press(t, a, ":"), "7")
	a = press(t, a, "V", "j", "j")
	screenIs(t, a, "visual")
	a = press(t, a, "o")
	if at, _, _ := a.helpers.Editing.Selected(a.ContextTree); at.StartLine != 7 || at.Line != 9 {
		t.Fatalf("o keeps the same lines: %+v", at)
	}
	a = press(t, a, "c")
	if a.Bad || a.Diff.Selecting {
		t.Fatalf("the range was not queued: %q", a.Notice)
	}
	if n, ok := presentation.NoteOf(*a.Review, "Sources/Catalog/MemoryCatalog.swift", "queued:4"); !ok || n.At.StartLine != 7 || n.At.Line != 9 {
		t.Fatalf("queued comment 4 shows on %+v", n.At)
	}
}

func TestACommentWhereNoneCanGoSaysWhyFirst(t *testing.T) {
	editor, seen := writer(t, "Never sent.")
	a := onBasket(t, editor)
	a = typed(t, press(t, a, "space", "f"), "Catalog/Catalog.swift")
	a = typed(t, press(t, a, ":"), "1")
	a = press(t, a, "c")
	if !a.Bad || a.Notice != "This file is not in the change, so a comment cannot go on it. Ask Claude about it instead." {
		t.Fatalf("the status line says %q", a.Notice)
	}
	if _, err := os.Stat(seen); err == nil {
		t.Fatal("the editor opened for a comment that cannot go anywhere")
	}
}

func TestSavingNothingSendsNothing(t *testing.T) {
	editor, _ := writer(t, "")
	a := onBasket(t, editor)
	a = press(t, typed(t, press(t, a, ":"), "3"), "c")
	if a.Notice != "Nothing was written, so nothing was sent." {
		t.Fatalf("the status line says %q", a.Notice)
	}
}

func TestARefusedEditKeepsItsTextAndSavesOverTheNewVersion(t *testing.T) {
	editor, _ := writer(t, "Keep the order the items came in.")
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30, editor)
	state := a.Review.State
	queue := *state.Queue
	queue.Members = append([]models.QueueMember{}, queue.Members...)
	queue.Members[0].Comments = append([]models.Comment{}, queue.Members[0].Comments...)
	queue.Members[0].Comments[0].Version = "stale1"
	queue.Members[0].Comments[0].Body = ""
	state.Queue = &queue
	a = a.with(a.helpers.Refresh.WithState(a.ContextTree, state))
	a = press(t, a, "0", "]", "]")
	for a.ReviewView.View.Items[a.ReviewView.View.Cursor].Kind != "comment" {
		a = press(t, a, "j")
	}
	a = press(t, a, "e")
	if !a.Bad || !strings.HasPrefix(a.Notice, "not changed: comment 1 was changed after this window showed it") {
		t.Fatalf("the status line says %q", a.Notice)
	}
	if a.Drafts["edit 1"] != "Keep the order the items came in." {
		t.Fatalf("the text was not kept: %q", a.Drafts)
	}
	a = withEditor(a, func(o *Options) { o.Editor = "true" })
	a = press(t, a, "e")
	if a.Bad || a.Notice != "Saved queued comment 1." {
		t.Fatalf("saving again over the new version: %q", a.Notice)
	}
	if _, ok := a.Drafts["edit 1"]; ok {
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
	if a.Notice != "Kept queued comment 2." {
		t.Fatalf("the status line says %q", a.Notice)
	}
	a = press(t, a, "d", "d", "y")
	if a.Notice != "Dropped queued comment 2." {
		t.Fatalf("the status line says %q", a.Notice)
	}
	if _, ok := presentation.NoteOf(*a.Review, "Sources/Basket/Basket.swift", "queued:2"); ok {
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
	if a.Notice != "Claude answered." {
		t.Fatalf("the status line says %q", a.Notice)
	}
	screenIs(t, a, "asked")
	follow, seen := writer(t, "Could it be passed in?")
	a = withEditor(a, follow)
	a = press(t, a, "]", "n", "]", "n")
	if got := noteKeyAt(a); got != "notehead conversation:1" {
		t.Fatalf("on %s", got)
	}
	a = press(t, a, "a")
	file, _ = os.ReadFile(seen)
	if !strings.Contains(string(file), "# A follow-up to Claude on Sources/Basket/Basket.swift, line 7.\n# The conversation so far:\n#\n#   You: Could this be passed in?\n") {
		t.Fatalf("the editor was given:\n%s", file)
	}
	if n, _ := presentation.NoteOf(*a.Review, "Sources/Basket/Basket.swift", "conversation:1"); len(n.Conversation.Messages) != 4 {
		t.Fatalf("the follow-up and its answer: %+v", n.Conversation.Messages)
	}
}

func TestClaudesAnswerBecomesTheComment(t *testing.T) {
	a := onBasket(t, func(o *Options) { o.Editor = "true" })
	a = press(t, a, "]", "n", "z", "a", "j", "j")
	if r := a.Diff.Rows[a.Diff.Cursor]; r.Part != "claude" {
		t.Fatalf("on %+v", r)
	}
	a = press(t, a, "c")
	if a.Bad || a.Notice != "Queued: A comment on Sources/Basket/Basket.swift, line 7, from Claude's answer" {
		t.Fatalf("the status line says %q", a.Notice)
	}
}

func TestALineSeveralPullRequestsShowAsksWhichFirst(t *testing.T) {
	a := onBasket(t)
	e := types.Editing{Purpose: "comment", At: models.Lines{Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 2}, Key: "comment", About: []string{"A comment on Sources/Basket/Basket.swift, line 2."}}
	a = send(t, a, types.Checked{E: e, Where: models.Where{Words: "Choose the pull request it goes on.", Choose: []models.Choice{{Pull: 21, Path: "Sources/Basket/Basket.swift", Line: 2}, {Pull: 22, Path: "Sources/Basket/Basket.swift", Line: 2}}}})
	if !a.MenuOpen() || len(a.Menu.Rows) != 2 || a.Menu.Title != "Choose the pull request it goes on." {
		t.Fatalf("the choice: %+v", a.Menu)
	}
	a = press(t, a, "j", "enter")
	if !strings.Contains(a.Notice, "pull=22") {
		t.Fatalf("the choice was not asked about again with its pull request: %q", a.Notice)
	}
}

func barSays(a App) string {
	return strings.TrimSpace(ansi.Strip(a.barLine()))
}

func TestTheBarSaysClaudeIsWorkingWithASpinnerUntilItAnswers(t *testing.T) {
	ticks := 0
	a := onBasket(t, func(o *Options) {
		o.Tick = func(time.Duration, func(time.Time) tea.Msg) tea.Cmd { ticks++; return nil }
	})
	question := filepath.Join(t.TempDir(), "question.md")
	if err := os.WriteFile(question, []byte("Why keep a search here?\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	at := models.Lines{Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 3}
	next, ask := a.Update(types.Edited{E: types.Editing{Purpose: "ask", At: at, Key: "ask " + at.Path}, File: question})
	a = next.(App)
	if got := barSays(a); !strings.HasSuffix(got, "Claude is working |") || ticks != 1 {
		t.Fatalf("while Claude answers, the bar says %q after %d ticks asked for", got, ticks)
	}
	a = send(t, a, types.Spun{})
	if got := barSays(a); !strings.HasSuffix(got, "Claude is working /") || ticks != 2 {
		t.Fatalf("a tick turns the spinner: %q after %d ticks", got, ticks)
	}
	a = press(t, drive(t, a, ask), "j")
	if got := barSays(a); strings.Contains(got, "Claude is working") {
		t.Fatalf("once Claude answered, the bar still says %q", got)
	}
	a = send(t, a, types.Spun{})
	if ticks != 2 {
		t.Fatalf("the spinner went on after the answer: %d ticks", ticks)
	}
}

func TestAQuestionWaitingFromTheOtherAppTurnsTheSpinnerToo(t *testing.T) {
	ticks := 0
	a := onBasket(t, func(o *Options) {
		o.Tick = func(time.Duration, func(time.Time) tea.Msg) tea.Cmd { ticks++; return nil }
	})
	st := a.Review.State
	st.Threads = append(append([]models.Conversation{}, st.Threads...), models.Conversation{
		ID: 9, Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 3, Messages: []models.Message{{From: "you", Text: "Who calls this?"}},
	})
	a = send(t, a, types.StateLoaded{State: st})
	if got := barSays(a); !strings.HasSuffix(got, "Claude is working |") || ticks != 1 {
		t.Fatalf("the bar says %q after %d ticks", got, ticks)
	}
	screenIs(t, press(t, a, "]", "n", "]", "n", "]", "n"), "asked-elsewhere")
}

func TestStoppingClaudeShowsWhatItDidAndSaysItStopped(t *testing.T) {
	a := onBasket(t)
	st := a.Review.State
	st.Threads = append(append([]models.Conversation{}, st.Threads...), models.Conversation{
		ID: 9, Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 3, Messages: []models.Message{{From: "you", Text: "Does anything test this?"}},
		Working: &models.Working{Steps: []models.Step{{Words: "Read Sources/Basket/Basket.swift"}, {Words: "Ran swift test"}}},
	})
	a = send(t, a, types.StateLoaded{State: st})
	if got := barSays(a); !strings.HasSuffix(got, "Claude is working: Ran swift test |") {
		t.Fatalf("the bar says %q", got)
	}
	a = press(t, a, "x")
	if a.Notice != "x stops Claude on a conversation it is answering" || !a.Bad {
		t.Fatalf("off a conversation, x says %q", a.Notice)
	}
	a = press(t, a, "]", "n")
	if got := noteKeyAt(a); got != "notehead conversation:9" {
		t.Fatalf("]n went to %s", got)
	}
	a = press(t, a, "z", "a")
	screenIs(t, a, "working")
	a = press(t, a, "x")
	if a.Notice != "Stopped Claude." || a.Bad {
		t.Fatalf("the status line says %q", a.Notice)
	}
	n, _ := presentation.NoteOf(*a.Review, "Sources/Basket/Basket.swift", "conversation:9")
	if m := n.Conversation.Messages; len(m) != 2 || m[1].Error != "you stopped it" || n.Conversation.Working != nil {
		t.Fatalf("the conversation after stopping: %+v", n.Conversation)
	}
	screenIs(t, a, "stopped")
	a = send(t, a, types.Changed{E: types.Editing{Purpose: "ask"}, Conversation: n.Conversation})
	if a.Notice != "Stopped Claude." || a.Bad {
		t.Fatalf("when the question stopped here comes back, the status line says %q", a.Notice)
	}
}

func TestTheSpinnerStaysWhenTheBarCutsClaudesStep(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 60, 24, func(o *Options) {
		o.Tick = func(time.Duration, func(time.Time) tea.Msg) tea.Cmd { return nil }
	})
	st := a.Review.State
	st.Threads = append(append([]models.Conversation{}, st.Threads...), models.Conversation{
		ID: 9, Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 3, Messages: []models.Message{{From: "you", Text: "Does anything test this?"}},
		Working: &models.Working{Steps: []models.Step{{Words: "Ran swift test --filter BasketTests --parallel --enable-code-coverage"}}},
	})
	a = send(t, a, types.StateLoaded{State: st})
	if got := barSays(a); !strings.HasSuffix(got, "… |") || !strings.Contains(got, "Claude is working: Ran swift") {
		t.Fatalf("the bar says %q", got)
	}
}
