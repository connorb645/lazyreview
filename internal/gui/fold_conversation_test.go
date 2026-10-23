package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/models"
)

// withThreeOnALine is the app with two more conversations on the line of
// conversation 1, Basket.swift line 7, each with its own question.
func withThreeOnALine(t *testing.T, a App) App {
	t.Helper()
	st := a.Review.State
	st.Threads = append([]models.Conversation{}, st.Threads...)
	for _, c := range []struct {
		id                int
		at, question, ans string
	}{{2, "2026-10-15T13:00:02.000Z", "Who else reads the search?", "Only the basket does."}, {3, "2026-10-15T13:00:03.000Z", "Could the basket keep its own?", "It could, at the cost of a second index."}} {
		st.Threads = append(st.Threads, models.Conversation{ID: c.id, Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 7, CreatedAt: c.at,
			Messages: []models.Message{{From: "you", Text: c.question}, {From: "claude", Text: c.ans}}})
	}
	return send(t, a, types.StateLoaded{State: st})
}

func reviewItemThread(a App) int {
	if v := a.ReviewView.View; v.Cursor < len(v.Items) && v.Items[v.Cursor].Kind == "conversation" {
		return v.Items[v.Cursor].Thread
	}
	return 0
}

func toReviewItem(t *testing.T, a App, thread int) App {
	t.Helper()
	a = press(t, a, "g", "g")
	for range len(a.ReviewView.View.Items) {
		if reviewItemThread(a) == thread {
			return a
		}
		a = press(t, a, "j")
	}
	t.Fatalf("no conversation %d in the Review view", thread)
	return a
}

func TestTheReviewViewKeepsItsCursorOnTheConversationItWasOn(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = toReviewItem(t, press(t, withThreeOnALine(t, a), "0"), 3)
	a = send(t, a, types.StateLoaded{State: a.Review.State})
	if got := reviewItemThread(a); got != 3 {
		t.Fatalf("after the review changed, the cursor is on conversation %d, not 3, on the same line", got)
	}
}

func TestZaFoldsAndUnfoldsEachConversationOfALineInTheReviewView(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = toReviewItem(t, press(t, withThreeOnALine(t, a), "0"), 2)
	if got := barSays(a); got != "Move: j k | Open: Enter | Delete: dd | Fold: za | Sections: ]] [[ | List: Esc | Keys: Space ?" {
		t.Fatalf("on a conversation, the bar says %q", got)
	}
	a = press(t, a, "z", "a")
	if got := reviewItemThread(a); got != 2 || !a.ReviewView.Opened["conversation:2"] || a.ReviewView.Folded["conversations"] {
		t.Fatalf("za on conversation 2 left the cursor on %d, opened %v and folded %v", got, a.ReviewView.Opened, a.ReviewView.Folded)
	}
	shown := ansi.Strip(a.View())
	if !strings.Contains(shown, "▾ Sources/Basket/Basket.swift:7") || !strings.Contains(shown, "You: Who else reads the search?") || !strings.Contains(shown, "Claude: Only the basket does.") {
		t.Fatalf("an open conversation does not show every message:\n%s", shown)
	}
	screenIs(t, a, "review-conversation-open")
	a = press(t, a, "j", "z", "a", "k", "z", "a")
	if got := reviewItemThread(a); got != 2 || a.ReviewView.Opened["conversation:2"] || !a.ReviewView.Opened["conversation:3"] {
		t.Fatalf("after unfolding 3 and folding 2, the cursor is on %d and %v are open", got, a.ReviewView.Opened)
	}
}

// withOneOffTheDiff is the app with a conversation on a file the review's
// page does not hold, as after a push took the file out of the pull request.
func withOneOffTheDiff(t *testing.T, a App) App {
	t.Helper()
	st := a.Review.State
	st.Threads = append(append([]models.Conversation{}, st.Threads...), models.Conversation{ID: 4, Path: "Sources/Basket/Pricing.swift", Side: "RIGHT", Line: 12, CreatedAt: "2026-10-15T13:00:04.000Z",
		Messages: []models.Message{{From: "you", Text: "Does the basket round its total?"}, {From: "claude", Text: "Yes, to the penny, in Pricing.swift."}}})
	return send(t, a, types.StateLoaded{State: st})
}

func TestNotesShowsAConversationOffTheDiffOpenInTheReviewView(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 80, 24)
	a = press(t, withOneOffTheDiff(t, a), "3")
	for range 10 {
		if selectedKey(a, context.NotesPanel) == "conversation:4" {
			break
		}
		a = press(t, a, "j")
	}
	if a.Screen() != context.ScreenReview || reviewItemThread(a) != 4 || !a.ReviewView.Opened["conversation:4"] || a.Notice != "" {
		t.Fatalf("on a conversation off the diff, the main panel shows %v on conversation %d, opened %v, and says %q", a.Screen(), reviewItemThread(a), a.ReviewView.Opened, a.Notice)
	}
	if got := barSays(a); got != "Open: Enter | Delete: dd | Fold: za | Panels: h l | Keys: Space ?" {
		t.Fatalf("at 80 columns, the bar says %q", got)
	}
	screenIs(t, a, "notes-conversation-off-the-diff")
	a = press(t, a, "enter")
	if a.Focus() != context.MainPanel || reviewItemThread(a) != 4 {
		t.Fatalf("Enter gave the keys to panel %d, on conversation %d", a.Focus(), reviewItemThread(a))
	}
	a = press(t, a, "enter")
	if a.Screen() != context.ScreenReview || a.ReviewView.Opened["conversation:4"] {
		t.Fatalf("Enter on a conversation off the diff went to %v, leaving it opened: %v", a.Screen(), a.ReviewView.Opened)
	}
	a = press(t, a, "enter")
	if !a.ReviewView.Opened["conversation:4"] {
		t.Fatal("Enter again does not open the conversation off the diff")
	}
}

func toNotesRow(t *testing.T, a App, key string) App {
	t.Helper()
	a = press(t, a, "3", "g", "g")
	for range len(a.Side(context.NotesPanel).Rows) {
		if selectedKey(a, context.NotesPanel) == key {
			return a
		}
		a = press(t, a, "j")
	}
	t.Fatalf("no row %s in the Notes panel", key)
	return a
}

func TestZaInTheNotesPanelFoldsAndUnfoldsTheConversationWhereItShows(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 80, 24)
	a = toNotesRow(t, withThreeOnALine(t, a), "conversation:2")
	open := func(a App, key string) bool {
		n, _ := presentation.NoteOf(*a.Review, "Sources/Basket/Basket.swift", key)
		return a.Diff.NoteOpen(n)
	}
	if a.Screen() != context.ScreenDiff || !open(a, "conversation:2") {
		t.Fatalf("the main panel shows %v, conversation 2 open: %v", a.Screen(), open(a, "conversation:2"))
	}
	if got := barSays(a); got != "Open: Enter | Delete: dd | Fold: za | Panels: h l | Keys: Space ?" {
		t.Fatalf("at 80 columns, on a conversation, the bar says %q", got)
	}
	a = press(t, a, "z", "a")
	if open(a, "conversation:2") || a.Focus() != context.NotesPanel || selectedKey(a, context.NotesPanel) != "conversation:2" || noteKeyAt(a) != "notehead conversation:2" {
		t.Fatalf("za left conversation 2 open: %v, the keys in panel %d on %s, the diff on %s", open(a, "conversation:2"), a.Focus(), selectedKey(a, context.NotesPanel), noteKeyAt(a))
	}
	a = press(t, a, "z", "a")
	if !open(a, "conversation:2") || open(a, "conversation:3") {
		t.Fatal("za again does not unfold conversation 2 alone")
	}
	a = toNotesRow(t, withOneOffTheDiff(t, a), "conversation:4")
	a = press(t, a, "z", "a")
	if a.Screen() != context.ScreenReview || a.ReviewView.Opened["conversation:4"] || reviewItemThread(a) != 4 {
		t.Fatalf("za on a conversation off the diff shows %v, opened %v, on conversation %d", a.Screen(), a.ReviewView.Opened, reviewItemThread(a))
	}
	a = press(t, a, "z", "a")
	if !a.ReviewView.Opened["conversation:4"] {
		t.Fatal("za again does not open the conversation off the diff")
	}
}

func TestTheBarOffersDeleteAndFoldOnEveryRowOfAConversationAt80Columns(t *testing.T) {
	a := toNote(t, onBasketAt(t, 80, 24), "conversation:1")
	a = press(t, a, "z", "o")
	for a.Diff.Rows[a.Diff.Cursor].Key == "conversation:1" {
		r := a.Diff.Rows[a.Diff.Cursor]
		if got := barSays(a); !strings.Contains(got, "Delete: dd") || !strings.Contains(got, "Fold: za") {
			t.Errorf("on the conversation's %s %s row, the bar says %q", r.Kind, r.Part, got)
		}
		a = press(t, a, "j")
	}
}

func TestEachFoldedConversationOfALineShowsTheStartOfItsQuestion(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {161, 48}} {
		a := press(t, withThreeOnALine(t, onBasketAt(t, size[0], size[1])), "z", "M", "z", "o", "]", "n")
		var heads []string
		for _, l := range strings.Split(ansi.Strip(a.View()), "\n") {
			if strings.Contains(l, "▸ Claude") {
				heads = append(heads, l)
			}
		}
		if len(heads) != 3 {
			t.Fatalf("at %d columns, %d folded conversations show:\n%s", size[0], len(heads), a.View())
		}
		for i, start := range []string{"· Could this", "· Who else", "· Could the b"} {
			if !strings.Contains(heads[i], start) {
				t.Errorf("at %d columns, folded conversation %d reads %q", size[0], i+1, heads[i])
			}
		}
		screenIs(t, a, fmt.Sprintf("conversations-folded-%dx%d", size[0], size[1]))
	}
}

func TestNotesOpensOnlyTheConversationAtItsCursorAndKeepsOneOpenedOnPurpose(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = withThreeOnALine(t, a)
	opens := func(a App) string {
		var open []string
		for _, key := range []string{"conversation:1", "conversation:2", "conversation:3"} {
			if n, _ := presentation.NoteOf(*a.Review, "Sources/Basket/Basket.swift", key); a.Diff.NoteOpen(n) {
				open = append(open, key)
			}
		}
		return strings.Join(open, " ")
	}
	for _, key := range []string{"conversation:1", "conversation:2", "conversation:3"} {
		a = toNotesRow(t, a, key)
		if got := opens(a); got != key {
			t.Fatalf("with the Notes panel on %s, %q are open", key, got)
		}
	}
	a = press(t, a, "z", "a", "z", "a", "k")
	if got := opens(a); got != "conversation:2 conversation:3" {
		t.Fatalf("after za twice on conversation 3, then k, %q are open", got)
	}
	a = press(t, a, "k")
	if got := opens(a); got != "conversation:1 conversation:3" {
		t.Fatalf("after k again, %q are open", got)
	}
}

func TestTheKeysListOfEachPlaceAConversationShowsSaysHowToFoldAndDeleteIt(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	for _, c := range []struct {
		place, fold string
		at          App
	}{
		{"the diff", "a note such as a conversation with Claude", toNote(t, onBasketAt(t, 100, 30), "conversation:1")},
		{"the Notes panel", "folds or unfolds the note", toNotesRow(t, a, "conversation:1")},
		{"the Review view", "folds or unfolds the conversation at the cursor", toReviewItem(t, press(t, a, "0"), 1)},
	} {
		help := map[string]string{}
		for _, r := range press(t, c.at, "space", "?").Menu.Rows {
			help[r.Left] = r.Right
		}
		fold := help["za"] + help["za zo zc"]
		if !strings.Contains(fold, c.fold) || !strings.Contains(help["dd"], "deletes the conversation with Claude") {
			t.Errorf("in %s, Space ? lists za as %q and dd as %q", c.place, fold, help["dd"])
		}
	}
}

func TestAConversationsHeadInTheReviewViewReadsWholeAt80Columns(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 80, 24)
	st := withOneOffTheDiff(t, a).Review.State
	st.Threads = append(st.Threads, models.Conversation{ID: 5, Path: "Sources/Basket/Checkout/Pricing/Rounding.swift", Side: "RIGHT", Line: 120, CreatedAt: "2026-10-15T13:00:05.000Z",
		Messages: []models.Message{{From: "you", Text: "Why round here?"}, {From: "claude", Text: "The total is shown to the penny."}}})
	a = press(t, send(t, a, types.StateLoaded{State: st}), "0")
	for id, want := range map[int]string{4: "▸ Sources/Basket/Pricing.swift:12  2 messages", 5: "▸ …/Pricing/Rounding.swift:120  2 messages"} {
		a = toReviewItem(t, a, id)
		head := ""
		for _, l := range a.ReviewView.View.Lines {
			if l.Item == a.ReviewView.View.Cursor {
				head = strings.TrimSpace(ansi.Strip(l.Text))
				break
			}
		}
		if head != want {
			t.Errorf("at 80 columns, conversation %d's head reads %q", id, head)
		}
	}
}

func TestAQuestionSentToClaudeIsNoDraftOfTheNextOnItsLine(t *testing.T) {
	editor, seen := writer(t, "Who else reads the search?")
	a := typed(t, press(t, onBasket(t, editor), ":"), "7")
	question := filepath.Join(t.TempDir(), "question.md")
	if err := os.WriteFile(question, []byte("Why keep a search here?\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	at := models.Lines{Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 7}
	key := "ask " + helpers.LinesKey(at)
	next, _ := a.Update(types.Edited{E: types.Editing{Purpose: "ask", At: at, Key: key}, File: question})
	a = next.(App)
	if draft, ok := a.Drafts[key]; ok {
		t.Fatalf("while Claude answers, the question sent is kept as a draft: %q", draft)
	}
	a = press(t, a, "a")
	if opened, err := os.ReadFile(seen); err != nil || strings.Contains(string(opened), "Why keep a search here?") {
		t.Fatalf("a second question on the line opens with the first in it: %q, %v", opened, err)
	}
	if got := a.Drafts[key]; got != "Who else reads the search?" || !a.Bad {
		t.Fatalf("a question that was refused keeps %q as its draft, and the bar says %q", got, a.Notice)
	}
}
