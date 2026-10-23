package gui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/gui/context"
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
	if got := barSays(a); got != "Open: Enter | Delete: dd | Panels: h l | Keys: Space ?" {
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
