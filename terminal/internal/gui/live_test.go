package gui

import (
	stdcontext "context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
)

const stack = "github.com/octo/widgets/21"

func changedElsewhere(t *testing.T) (answer, string) {
	t.Helper()
	for key, a := range recorded(t) {
		if strings.HasPrefix(key, "POST /api/reviews/"+stack+"/comments ") && strings.Contains(key, "Name the search for what it finds.") {
			var queued struct{ State json.RawMessage }
			if err := json.Unmarshal(a.Body, &queued); err != nil {
				t.Fatal(err)
			}
			var state server.State
			if err := json.Unmarshal(queued.State, &state); err != nil {
				t.Fatal(err)
			}
			return answer{Status: 200, Body: queued.State}, state.Version
		}
	}
	t.Fatal("no recorded comment queued on Basket.swift")
	return answer{}, ""
}

func TestAChangeElsewhereShowsAtOnce(t *testing.T) {
	state, version := changedElsewhere(t)
	a, _, changed := openedWith(t, stack, 100, 30)
	changed.set("GET /api/reviews/"+stack, state)
	a = press(t, onBasketFrom(t, a), ":")
	a = typed(t, a, "3")
	folded := map[string]bool{"Sources/Catalog/MemoryCatalog.swift": true}
	a.Diff.Folded = folded
	a = send(t, a, types.LiveNotice{ID: stack, Notice: server.Notice{State: a.Review.State.Version}})
	if _, ok := presentation.NoteOf(*a.Review, "Sources/Basket/Basket.swift", "queued:3"); ok {
		t.Fatal("a notice of the state the app holds read it again")
	}
	a = send(t, a, types.LiveNotice{ID: stack, Notice: server.Notice{State: version}})
	if _, ok := presentation.NoteOf(*a.Review, "Sources/Basket/Basket.swift", "queued:3"); !ok {
		t.Fatal("the comment queued elsewhere does not show")
	}
	if cursorAt(a) != "Sources/Basket/Basket.swift RIGHT 3" || !a.Diff.Folded["Sources/Catalog/MemoryCatalog.swift"] {
		t.Fatalf("the place and the folds moved: %s %v", cursorAt(a), a.Diff.Folded)
	}
	screenIs(t, a, "live-state")
}

func onBasketFrom(t *testing.T, a App) App {
	t.Helper()
	return typed(t, press(t, a, "space", "f"), "Basket/Basket")
}

func TestANewBuildLoadsInPlaceUnlessADraftIsOpen(t *testing.T) {
	page := recorded(t)["GET /api/reviews/"+stack+"/page"]
	var drawn map[string]json.RawMessage
	if err := json.Unmarshal(page.Body, &drawn); err != nil {
		t.Fatal(err)
	}
	drawn["build"] = json.RawMessage(`"0123456789ab"`)
	body, _ := json.Marshal(drawn)
	a, _, changed := openedWith(t, stack, 100, 30)
	changed.set("GET /api/reviews/"+stack+"/page", answer{Status: 200, Body: body})
	a = press(t, a, "space", "m", "enter", "enter", "j")
	a.Drafts = map[string]string{"comment x": "Half written."}
	a = send(t, a, types.LiveNotice{ID: stack, Notice: server.Notice{Build: "0123456789ab"}})
	if a.Review.Page.Build == "0123456789ab" || a.Notice != "This review was loaded again. R loads it here; your drafts stay." {
		t.Fatalf("with a draft open: %s %q", a.Review.Page.Build, a.Notice)
	}
	a.Drafts = nil
	a = send(t, a, types.LiveNotice{ID: stack, Notice: server.Notice{Build: "0123456789ab"}})
	if a.Review.Page.Build != "0123456789ab" || a.Notice != "Loaded again." {
		t.Fatalf("with no draft open: %s %q", a.Review.Page.Build, a.Notice)
	}
	if a.Screen() != context.ScreenDiagram {
		t.Fatalf("the reload left the diagram for %v", a.Screen())
	}
	focusedOn(t, a, "Basket")
}

// R in the Diagram panel names its boxes as the review now draws them,
// as the main panel does, not as they were drawn before.
func TestRNamesTheDiagramPanelsBoxesAfresh(t *testing.T) {
	page := recorded(t)["GET /api/reviews/"+stack+"/page"]
	var drawn map[string]any
	if err := json.Unmarshal(page.Body, &drawn); err != nil {
		t.Fatal(err)
	}
	drawn["build"] = "0123456789ab"
	component := drawn["model"].(map[string]any)["components"].([]any)[0].(map[string]any)
	component["packages"].([]any)[0].(map[string]any)["title"] = "Shelves"
	body, _ := json.Marshal(drawn)
	a, _, changed := openedWith(t, stack, 100, 30)
	a = press(t, a, "4", "enter", "enter")
	changed.set("GET /api/reviews/"+stack+"/page", answer{Status: 200, Body: body})
	changed.set("POST /api/reviews/"+stack+"/refresh {}", answer{Status: 200, Body: json.RawMessage(`{"id":"` + stack + `","notes":[]}`)})
	a = press(t, a, "R")
	var titles []string
	for _, row := range a.Side(context.DiagramPanel).Rows {
		titles = append(titles, row.Title)
	}
	if a.Notice != "Loaded again." || !slices.Contains(titles, "Shelves") || !strings.Contains(a.View(), "Shelves") {
		t.Fatalf("after %q the Diagram panel's boxes are %q", a.Notice, titles)
	}
}

func TestRLoadsTheReviewAgainInPlace(t *testing.T) {
	a := onGuide(t)
	a = press(t, a, "R")
	if a.Notice != "Loaded again." || a.Bad {
		t.Fatalf("the status line says %q", a.Notice)
	}
	if a.Screen() != context.ScreenDiff || !a.Diff.Rendered["docs/guide.md"] || cursorAt(a) != "docs/guide.md file" {
		t.Fatalf("R left the rendered guide: %v %s", a.Screen(), cursorAt(a))
	}
}

func TestAStoppedStreamSaysSo(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = send(t, a, types.LiveStopped{ID: stack, Err: errString("the server ended the stream of changes")})
	if !a.Bad || a.Notice != "Stopped hearing changes made elsewhere: the server ended the stream of changes. R loads the review again." {
		t.Fatalf("the status line says %q", a.Notice)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestListeningComesBackAfterTheStreamEnds(t *testing.T) {
	connected := 0
	client := servedBy(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		connected++
		fmt.Fprintf(w, "retry: 1000\n\ndata: {\"state\":\"v%d\"}\n\n", connected)
	}))
	ctx, stop := stdcontext.WithCancel(stdcontext.Background())
	defer stop()
	heard := helpers.Listen(ctx, client, stack)
	for _, want := range []string{"v1", "v2"} {
		select {
		case msg := <-heard:
			if n, ok := msg.(types.LiveNotice); !ok || n.Notice.State != want {
				t.Fatalf("heard %+v, not %s", msg, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("nothing heard for %s", want)
		}
	}
}

func TestAQuestionAskedElsewhereShowsAsWaiting(t *testing.T) {
	a := onBasket(t)
	state := a.Review.State
	state.Threads = append([]server.Conversation{}, state.Threads...)
	state.Threads = append(state.Threads, server.Conversation{ID: 9, Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 3, Messages: []server.Message{{From: "you", Text: "Why here?"}}})
	a = a.with(a.helpers.Refresh.WithState(a.ContextTree, state))
	n, _ := presentation.NoteOf(*a.Review, "Sources/Basket/Basket.swift", "conversation:9")
	if head, _ := presentation.NoteHead(a.c.Tr, a.controllers.Labels(context.DiffKey), *a.Review, n, false); head != "▸ Claude · 1 message · line 3 · waiting for Claude" {
		t.Fatalf("the conversation's head reads %q", head)
	}
}
