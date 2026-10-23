package gui

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/models"
)

const deleteOne = "DELETE /api/reviews/github.com/octo/widgets/21/threads/1"

func conversationShows(a App, id int) bool {
	return slices.ContainsFunc(a.Review.State.Threads, func(c models.Conversation) bool { return c.ID == id })
}

func stateAnswered(t *testing.T, st models.State) answer {
	t.Helper()
	body, err := json.Marshal(map[string]any{"state": st})
	if err != nil {
		t.Fatal(err)
	}
	return answer{Status: 200, Body: body}
}

func toNote(t *testing.T, a App, key string) App {
	t.Helper()
	a = press(t, a, "g", "g")
	for range 20 {
		if noteKeyAt(a) == "notehead "+key {
			return a
		}
		a = press(t, a, "]", "n")
	}
	t.Fatalf("no note %s in the diff", key)
	return a
}

func TestDeletingAConversationInTheDiffAsksFirstNamingIt(t *testing.T) {
	a := onBasket(t)
	a = press(t, a, "]", "n")
	if got := noteKeyAt(a); got != "notehead conversation:1" {
		t.Fatalf("]n went to %s", got)
	}
	a = press(t, a, "d", "d")
	if c := a.Confirmation.Conversation; !a.Has(context.ConfirmationKey) || c == nil || c.ID != 1 {
		t.Fatalf("dd on a conversation asks about %+v", a.Confirmation)
	}
	screenIs(t, a, "delete-asks")
	if got := barSays(a); got != "Delete: y | Keep: any other key" {
		t.Fatalf("the bar says %q", got)
	}
	a = press(t, a, "n")
	if a.Notice != "Kept the conversation on Basket.swift, line 7." || a.Bad || !conversationShows(a, 1) {
		t.Fatalf("n keeps it, and the status line says %q", a.Notice)
	}
	a = press(t, a, "z", "a", "j", "j")
	if r := a.Diff.Rows[a.Diff.Cursor]; r.Kind != "notetext" || r.Key != "conversation:1" {
		t.Fatalf("on %+v", r)
	}
	a = press(t, a, "d", "d", "y")
	if a.Notice != "Deleted the conversation on Basket.swift, line 7." || a.Bad {
		t.Fatalf("the status line says %q", a.Notice)
	}
	if _, ok := presentation.NoteOf(*a.Review, "Sources/Basket/Basket.swift", "conversation:1"); ok || conversationShows(a, 1) {
		t.Fatal("the deleted conversation still shows")
	}
	if _, open := a.Diff.Notes["conversation:1"]; open {
		t.Fatal("the deleted conversation's fold is kept for the next conversation 1")
	}
	if got := cursorAt(a); got != "Sources/Basket/Basket.swift RIGHT 7" {
		t.Fatalf("the cursor went to %s, not the line the conversation was on", got)
	}
	fromAnswer := false
	for _, m := range a.Review.State.Queue.Members {
		for _, c := range m.Comments {
			fromAnswer = fromAnswer || c.Body == "It could: the basket only reads from it."
		}
	}
	if !fromAnswer {
		t.Fatal("the comment made from Claude's answer went with the conversation")
	}
	screenIs(t, a, "deleted")
}

func TestDeletingAConversationFromTheNotesPanel(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "3", "j", "j")
	if got := selectedKey(a, context.NotesPanel); got != "conversation:1" {
		t.Fatalf("on %s", got)
	}
	a = press(t, a, "d", "d")
	if c := a.Confirmation.Conversation; c == nil || c.ID != 1 {
		t.Fatalf("dd in the Notes panel asks about %+v", a.Confirmation)
	}
	a = press(t, a, "y")
	if a.Notice != "Deleted the conversation on Basket.swift, line 7." || a.Side(context.NotesPanel).Index("conversation:1") >= 0 {
		t.Fatalf("the status line says %q, and the row stays", a.Notice)
	}
	if a.Focus() != context.NotesPanel {
		t.Fatalf("the keys went to panel %d", a.Focus())
	}
	if got, chosen := noteKeyAt(a), selectedKey(a, context.NotesPanel); got != "notehead "+chosen {
		t.Fatalf("the main panel shows %s, not the note chosen in its place, %s", got, chosen)
	}
}

func TestDeletingAConversationFromTheReviewView(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "0")
	for a.ReviewView.View.Items[a.ReviewView.View.Cursor].Kind != "conversation" {
		a = press(t, a, "j")
	}
	if got := barSays(a); got != "Move: j k | Open: Enter | Delete: dd | Fold: za | Sections: ]] [[ | List: Esc | Keys: Space ?" {
		t.Fatalf("on a conversation, the bar says %q", got)
	}
	a = press(t, a, "d", "d")
	if c := a.Confirmation.Conversation; c == nil || c.ID != 1 {
		t.Fatalf("dd in the Review view asks about %+v", a.Confirmation)
	}
	a = press(t, a, "y")
	if a.Notice != "Deleted the conversation on Basket.swift, line 7." || conversationShows(a, 1) {
		t.Fatalf("the status line says %q", a.Notice)
	}
}

func TestDeletingAConversationClaudeIsAnsweringSaysItStopsItFirst(t *testing.T) {
	a, _, changed := openedWith(t, stack, 100, 30)
	a = onBasketFrom(t, a)
	st := a.Review.State
	before := st
	st.Threads = append(append([]models.Conversation{}, st.Threads...), models.Conversation{
		ID: 9, Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 3, CreatedAt: "2026-10-15T13:00:09.000Z", Messages: []models.Message{{From: "you", Text: "Does anything test this?"}},
		Working: &models.Working{Steps: []models.Step{{Words: "Ran swift test"}}},
	})
	a = send(t, a, types.StateLoaded{State: st})
	a = press(t, a, "]", "n")
	if got := noteKeyAt(a); got != "notehead conversation:9" {
		t.Fatalf("]n went to %s", got)
	}
	a = press(t, a, "d", "d")
	screenIs(t, a, "delete-asks-working")
	changed.set("DELETE /api/reviews/github.com/octo/widgets/21/threads/9", stateAnswered(t, before))
	next, del := a.Update(keyMsg("y"))
	a = next.(App)
	if a.Notice != "Stopping Claude, then deleting…" {
		t.Fatalf("while it deletes, the status line says %q", a.Notice)
	}
	stopped := st.Threads[len(st.Threads)-1]
	stopped.Messages = append(stopped.Messages, models.Message{From: "claude", Error: "you stopped it"})
	stopped.Working = nil
	a = send(t, a, types.Changed{E: types.Editing{Purpose: "ask"}, Conversation: &stopped})
	if n, _ := presentation.NoteOf(*a.Review, "Sources/Basket/Basket.swift", "conversation:9"); len(n.Conversation.Messages) != 1 || a.Notice != "Stopping Claude, then deleting…" {
		t.Fatalf("the answer that stopped came back into the conversation being deleted: %+v, %q", n.Conversation, a.Notice)
	}
	a = drive(t, a, del)
	if a.Notice != "Deleted the conversation on Basket.swift, line 3." || conversationShows(a, 9) {
		t.Fatalf("the status line says %q", a.Notice)
	}
	a = send(t, a, types.Changed{E: types.Editing{Purpose: "ask"}, Conversation: &stopped, Err: models.Refused("there is no conversation 9 on " + stack)})
	if a.Notice != "Deleted the conversation on Basket.swift, line 3." || a.Bad || conversationShows(a, 9) {
		t.Fatalf("the question's late answer brought it back, or says %q", a.Notice)
	}
}

func TestARefusedDeleteSaysWhyAndKeepsTheConversation(t *testing.T) {
	a, _, changed := openedWith(t, stack, 100, 30)
	a = press(t, onBasketFrom(t, a), "]", "n")
	changed.set(deleteOne, answer{Status: 422, Body: json.RawMessage(`{"error":"not deleted: Claude is still answering it and did not stop"}`)})
	a = press(t, a, "d", "d", "y")
	if a.Notice != "not deleted: Claude is still answering it and did not stop" || !a.Bad || !conversationShows(a, 1) || len(a.Deleted) != 0 {
		t.Fatalf("the status line says %q, the conversations deleted here are %v", a.Notice, a.Deleted)
	}
	c := a.Review.State.Threads[0]
	c.Messages = append(append([]models.Message{}, c.Messages...), models.Message{From: "claude", Error: "you stopped it"})
	a = send(t, a, types.Changed{E: types.Editing{Purpose: "ask"}, Conversation: &c})
	if n, _ := presentation.NoteOf(*a.Review, "Sources/Basket/Basket.swift", "conversation:1"); len(n.Conversation.Messages) != len(c.Messages) {
		t.Fatal("an answer to the conversation kept does not show")
	}
	st := a.Review.State
	st.Threads = append(append([]models.Conversation{}, st.Threads...), models.Conversation{ID: 7, Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 3, Messages: []models.Message{{From: "you", Text: "Gone?"}, {From: "claude", Text: "Yes."}}})
	a = send(t, a, types.StateLoaded{State: st})
	a = press(t, toNote(t, a, "conversation:7"), "d", "d", "y")
	if a.Notice != "not deleted: there is no conversation 7 on github.com/octo/widgets/21" || !a.Bad || conversationShows(a, 7) {
		t.Fatalf("deleting one already gone says %q", a.Notice)
	}
}

func TestDdOffAQueuedCommentOrAConversationSaysWhatItDoes(t *testing.T) {
	a := onBasket(t)
	a = press(t, a, "d", "d")
	if a.Notice != "dd drops the queued comment, or deletes the conversation with Claude, at the cursor" || !a.Bad || a.Has(context.ConfirmationKey) {
		t.Fatalf("off a note, dd says %q", a.Notice)
	}
}
