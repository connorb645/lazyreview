package loader

import (
	"os"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/review"
	"github.com/connorb645/lazyreview/internal/store"
)

// asked keeps conversations with Claude in a review, as the app keeps them
// once Claude was asked.
func asked(t *testing.T, reviews *store.ReviewStore, ref models.ReviewRef, conversations ...models.Conversation) {
	t.Helper()
	err := reviews.Change([]models.ReviewRef{ref}, func() ([]store.FileWrite, error) {
		var writes []store.FileWrite
		for _, c := range conversations {
			c.CreatedAt, c.Session = "2026-10-23T09:30:00.000Z", "session-1"
			c.Messages = []models.Message{{From: "you", Text: "Why?"}, {From: "claude", Text: "Because."}}
			write, err := reviews.ThreadWrite(ref, c)
			if err != nil {
				return nil, err
			}
			writes = append(writes, write)
		}
		return writes, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// codeAt is the text of a conversation's last line as the review's page now
// draws it, and the number of the stack's member that changed that line,
// or 0; and false when the page cannot open the conversation's file.
func codeAt(t *testing.T, reviews *store.ReviewStore, ref models.ReviewRef, c models.Conversation) (string, int, bool) {
	t.Helper()
	tr := i18n.EnglishTranslationSet()
	drawn, err := reviews.ReadPage(ref)
	if err != nil {
		t.Fatal(err)
	}
	file, ok := review.FileView(tr, drawn.Page, c.Path)
	if !ok || c.Line < 1 || c.Line > len(file.Whole) {
		return "", 0, false
	}
	member := 0
	if drawn.Page.Stack != nil {
		member = review.LineEntry(tr, drawn.Page.Stack.Lines, c.Path, c.Line, c.Side).Pull
	}
	return file.Whole[c.Line-1], member, true
}

func conversationsOf(t *testing.T, reviews *store.ReviewStore, ref models.ReviewRef) map[int]models.Conversation {
	t.Helper()
	threads, err := reviews.Threads(ref)
	if err != nil {
		t.Fatal(err)
	}
	out := map[int]models.Conversation{}
	for _, c := range threads {
		out[c.ID] = c
	}
	return out
}

func TestConversationsOfABranchStayOnTheirCodeAsItGrowsIntoAStackAndLoadsAgain(t *testing.T) {
	r, reviews, branches := stackBench(t)
	r.Commit("stack/basket", map[string]string{"src/search.ts": "// Search the catalog by name.\n\nexport function find(text: string) {\n  return text;\n}\n"}, "Say what search does")
	r.Git("rebase", "-q", "stack/basket", "stack/total")
	r.Checkout("main")
	alone := stackLoad(t, branches, r, "stack/search")
	asked(t, reviews, alone.Ref,
		models.Conversation{ID: 1, Path: "src/search.ts", Side: "RIGHT", Line: 2},
		models.Conversation{ID: 2, Path: "src/search.ts", Side: "RIGHT", Line: 2},
		models.Conversation{ID: 3, Path: "src/search.ts", Side: "RIGHT", StartLine: 1, Line: 3},
		models.Conversation{ID: 4, Path: "src/catalog.ts", Side: "RIGHT", Line: 1},
	)
	if _, _, ok := codeAt(t, reviews, alone.Ref, conversationsOf(t, reviews, alone.Ref)[4]); ok {
		t.Fatal("a conversation on a file the change leaves out opens before the review is loaded again")
	}
	code := map[int]string{1: "  return text;", 2: "  return text;", 3: "}", 4: "export const items: string[] = [];"}
	check := func(when string, lines map[int][2]int) {
		t.Helper()
		kept := conversationsOf(t, reviews, alone.Ref)
		if len(kept) != 4 {
			t.Fatalf("%s, the review holds %d conversations", when, len(kept))
		}
		for id, want := range lines {
			c := kept[id]
			shown, member, ok := codeAt(t, reviews, alone.Ref, c)
			if !ok || shown != code[id] || c.StartLine != want[0] || c.Line != want[1] || len(c.Messages) != 2 || c.Session != "session-1" {
				t.Errorf("%s, conversation %d is on %s:%d-%d, reading %q, opens %v: %+v", when, id, c.Path, c.StartLine, c.Line, shown, ok, c)
			}
			if c.Path == "src/search.ts" && member > 1 {
				t.Errorf("%s, conversation %d is on a line of member %d, not stack/search's", when, id, member)
			}
		}
	}
	stackLoad(t, branches, r, "stack/search")
	check("on the branch alone, loaded again", map[int][2]int{1: {0, 2}, 2: {0, 2}, 3: {1, 3}, 4: {0, 1}})
	moved := map[int][2]int{1: {0, 4}, 2: {0, 4}, 3: {3, 5}, 4: {0, 1}}
	stackLoad(t, branches, r, "stack/search", "stack/basket", "stack/total")
	check("in the stack", moved)
	stackLoad(t, branches, r, "stack/search")
	check("loaded again from the bottom branch alone", moved)
	ref, err := models.ParseBranchRef(i18n.EnglishTranslationSet(), alone.Ref.ID())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := branches.Refresh(ref); err != nil {
		t.Fatal(err)
	}
	check("after R", moved)
}

func TestAConversationOnALineABranchAboveChangesStaysWhereTheCodeWent(t *testing.T) {
	r, reviews, branches := stackBench(t)
	r.Commit("stack/basket", map[string]string{"src/search.ts": "export function find(text: string) {\n  return text.trim();\n}\n"}, "Trim the search")
	r.Checkout("main")
	alone := stackLoad(t, branches, r, "stack/search")
	asked(t, reviews, alone.Ref, models.Conversation{ID: 1, Path: "src/search.ts", Side: "RIGHT", Line: 2})
	stackLoad(t, branches, r, "stack/search", "stack/basket")
	c := conversationsOf(t, reviews, alone.Ref)[1]
	shown, member, ok := codeAt(t, reviews, alone.Ref, c)
	if !ok || c.Line != 2 || shown != "  return text.trim();" || member != 2 {
		t.Errorf("the conversation is on line %d, reading %q, changed by member %d, opens %v", c.Line, shown, member, ok)
	}
}

func TestMovingConversationsWritesNoneWhoseLinesStayed(t *testing.T) {
	r, reviews, branches := stackBench(t)
	alone := stackLoad(t, branches, r, "stack/search")
	asked(t, reviews, alone.Ref, models.Conversation{ID: 1, Path: "src/search.ts", Side: "RIGHT", Line: 2})
	path := reviews.Home().Review(alone.Ref).Thread(1)
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	stackLoad(t, branches, r, "stack/search", "stack/basket")
	if after, err := os.Stat(path); err != nil || !after.ModTime().Equal(before.ModTime()) {
		t.Errorf("a conversation whose line stayed was written again: %v", err)
	}
}
