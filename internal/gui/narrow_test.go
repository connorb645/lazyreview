package gui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/models"
)

func onBasketAt(t *testing.T, width, height int, set ...func(*Options)) App {
	t.Helper()
	a, _ := opened(t, "github.com/octo/widgets/21", width, height, set...)
	a = typed(t, press(t, a, "space", "f"), "Basket/Basket")
	if a.Screen() != context.ScreenDiff || cursorAt(a) != "Sources/Basket/Basket.swift file" {
		t.Fatalf("the finder opened %v at %s", a.Screen(), cursorAt(a))
	}
	return a
}

func rowWith(a App, words string) string {
	for _, l := range strings.Split(ansi.Strip(a.View()), "\n") {
		if strings.Contains(l, words) {
			return l
		}
	}
	return ""
}

func TestAtEightyColumnsTheBarSaysWhatWasQueuedInFull(t *testing.T) {
	a := onBasketAt(t, 80, 24, func(o *Options) { o.Editor = "true" })
	a = press(t, a, "]", "n", "z", "a", "j", "j")
	a = press(t, a, "c")
	if got := barSays(a); !strings.HasSuffix(got, "Queued Claude's answer on Basket.swift, line 7.") {
		t.Errorf("at 80 columns the bar says %q", got)
	}
	a = a.with(a.Said("Deleted the conversation on MemoryCatalog.swift, lines 7-9.", false))
	if got := barSays(a); !strings.HasPrefix(got, "Keys: Space ? ") || !strings.HasSuffix(got, " Deleted the conversation on MemoryCatalog.swift, lines 7-9.") {
		t.Errorf("a notice that fits beside the last key is cut: %q", got)
	}
}

func TestAtEightyColumnsANotesHeadSaysItsStateInFull(t *testing.T) {
	a := onBasketAt(t, 80, 24)
	st := a.Review.State
	st.Threads = append(append([]models.Conversation{}, st.Threads...), models.Conversation{
		ID: 9, Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 11, Messages: []models.Message{{From: "you", Text: "Does anything test the total?"}},
	})
	a = send(t, a, types.StateLoaded{State: st})
	a = toNote(t, a, "conversation:9")
	if got := rowWith(a, "Claude · 1 message"); !strings.HasSuffix(strings.TrimRight(strings.TrimSuffix(got, "│"), " "), "▸ Claude · 1 message · waiting") {
		t.Errorf("at 80 columns the waiting conversation's head reads %q", got)
	}
	a = toNote(t, a, "queued:2")
	if got := rowWith(a, "Queued 2"); !strings.HasSuffix(strings.TrimRight(strings.TrimSuffix(got, "│"), " "), "▾ Queued 2 → #22") {
		t.Errorf("at 80 columns the queued comment's head reads %q", got)
	}
}
