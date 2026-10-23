package gui

import (
	"testing"

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
