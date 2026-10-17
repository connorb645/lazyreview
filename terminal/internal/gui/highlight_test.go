package gui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
)

func filterIs(t *testing.T, a App, want string) {
	t.Helper()
	if got := fmt.Sprint(a.Filter); got != want {
		t.Fatalf("the filter is %s, not %s", got, want)
	}
}

func rowKeys(a App, n int) []string {
	var keys []string
	for _, r := range a.Side(n).Rows {
		if !r.Heading {
			keys = append(keys, r.Key)
		}
	}
	return keys
}

func onThreeStack(t *testing.T) App {
	t.Helper()
	a, _ := opened(t, stack, 100, 30)
	d := *a.Review
	stacked := *d.Page.Stack
	stacked.Members = append(append([]server.StackMember{}, stacked.Members...), server.StackMember{Pull: 23, Head: "f00d"})
	d.Page.Stack = &stacked
	d.State.Members = append(append([]server.Member{}, d.State.Members...), server.Member{Pull: 23, Title: "Price the basket", Color: 3})
	a.Review = &d
	return a.relaid()
}

func TestVHighlightsAPullRequestAndJAndKGrowOrShrinkIt(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "v")
	filterIs(t, a, "[21]")
	if !strings.Contains(a.View(), "VISUAL · 1 of 2─╯") {
		t.Fatalf("the Review panel's foot does not say VISUAL:\n%s", a.View())
	}
	for _, key := range rowKeys(a, context.FilesPanel) {
		if key == "file:Sources/Basket/Basket.swift" || key == "file:web/src/cart.ts" {
			t.Fatalf("the Files panel shows %s, which only #22 changed", key)
		}
	}
	if got := strings.Join(rowKeys(a, context.NotesPanel), " "); got != "queued:1 thread:T1" {
		t.Fatalf("the Notes panel shows %s, not only #21's notes", got)
	}
	screenIs(t, a, "highlight-one")
	a = press(t, a, "j")
	filterIs(t, a, "[21 22]")
	if got := strings.Join(rowKeys(a, context.NotesPanel), " "); got != "queued:1 queued:2 conversation:1 thread:T1 thread:T2" {
		t.Fatalf("with both highlighted the Notes panel shows %s", got)
	}
	screenIs(t, a, "highlight-both")
	a = press(t, a, "k")
	filterIs(t, a, "[21]")
	a = press(t, a, "k")
	filterIs(t, a, "[21]")
}

func TestTheHighlightGrowsUpAndDownAThreePullRequestStack(t *testing.T) {
	a := press(t, onThreeStack(t), "G", "v")
	filterIs(t, a, "[23]")
	a = press(t, a, "k")
	filterIs(t, a, "[22 23]")
	a = press(t, a, "k")
	filterIs(t, a, "[21 22 23]")
	a = press(t, a, "j", "j")
	filterIs(t, a, "[23]")
	a = press(t, a, "g", "g")
	filterIs(t, a, "[21 22 23]")
	if !strings.Contains(send(t, a, tea.WindowSizeMsg{Width: 80, Height: 24}).View(), "─Review - #21-#23─") {
		t.Fatalf("at 80 columns the title does not name the highlight as #21-#23:\n%s", send(t, a, tea.WindowSizeMsg{Width: 80, Height: 24}).View())
	}
	a = press(t, a, "v", "j")
	filterIs(t, a, "[21 22]")
}

func TestTheHighlightStaysWhenTheKeysLeaveTheReviewPanel(t *testing.T) {
	for _, keys := range [][]string{{"2"}, {"3"}, {"4"}, {"h"}, {"l"}, {"0"}, {"enter"}} {
		a, _ := opened(t, stack, 100, 30)
		a = press(t, a, "v", "j")
		a = press(t, a, keys...)
		filterIs(t, a, "[21 22]")
		if a.GrowFrom != 0 || strings.Contains(a.View(), "VISUAL ·") {
			t.Fatalf("after %v the highlight still grows", keys)
		}
		a = press(t, a, "j", "k")
		filterIs(t, a, "[21 22]")
	}
}

func TestBackInTheReviewPanelJAndKMoveAndEscShowsTheWholeStack(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "j", "v", "2", "1")
	filterIs(t, a, "[22]")
	a = press(t, a, "k")
	if got := selectedKey(a, context.ReviewPanel); got != "pull:21" {
		t.Fatalf("k back in the Review panel moved to %s", got)
	}
	filterIs(t, a, "[22]")
	screenIs(t, a, "highlight-kept")
	a = press(t, a, "esc")
	filterIs(t, a, "[]")
	if a.Notice != "" {
		t.Fatalf("Esc that shows the whole stack says %q", a.Notice)
	}
	a = press(t, a, "v")
	filterIs(t, a, "[21]")
	a = press(t, a, "esc")
	filterIs(t, a, "[]")
	if a.GrowFrom != 0 {
		t.Fatal("Esc left the highlight growing")
	}
	a = press(t, a, "esc")
	if a.Notice != "ZZ or :q quits" {
		t.Fatalf("Esc with nothing highlighted says %q", a.Notice)
	}
}

func TestBracketPSetsTheHighlightToOnePullRequestOrAll(t *testing.T) {
	a := press(t, onThreeStack(t), "j", "v", "j")
	filterIs(t, a, "[22 23]")
	a = press(t, a, "[", "p")
	filterIs(t, a, "[21]")
	if a.GrowFrom != 0 {
		t.Fatal("[p left the highlight growing")
	}
	if !strings.Contains(a.View(), "[1]─Review - #21─") {
		t.Fatalf("with no colour the title names only the pull request highlighted:\n%s", a.View())
	}
	a = press(t, a, "]", "p")
	filterIs(t, a, "[22]")
	a = press(t, a, "v", "j", "]", "p")
	filterIs(t, a, "[]")
	a = press(t, a, "[", "p")
	filterIs(t, a, "[23]")
}

func TestVDoesNothingOnOnePullRequest(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	before := a.View()
	a = press(t, a, "v")
	filterIs(t, a, "[]")
	if a.GrowFrom != 0 || a.View() != before {
		t.Fatalf("v changed a review of one pull request:\n%s", a.View())
	}
}

func TestTheHighlightIsKeptAcrossAReload(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "v", "j", "2")
	a = send(t, a, types.ReviewReloaded{Data: *a.Review})
	filterIs(t, a, "[21 22]")
	d := *a.Review
	stacked := *d.Page.Stack
	stacked.Members = stacked.Members[:1]
	d.Page.Stack = &stacked
	a = send(t, a, types.ReviewReloaded{Data: d})
	filterIs(t, a, "[21]")
}

func TestTheReviewViewShowsOnlyWhatIsHighlighted(t *testing.T) {
	a, _ := opened(t, stack, 100, 60)
	a = press(t, a, "j", "v")
	view := a.View()
	for _, gone := range []string{"Keep the order items came in.", "Should this keep the order"} {
		if strings.Contains(view, gone) {
			t.Fatalf("with #22 highlighted the Review view shows #21's %q", gone)
		}
	}
	for _, kept := range []string{"Showing #22 only. Esc in the Review panel shows the whole", "Why one?", "widgets  2 files +4 −1"} {
		if !strings.Contains(view, kept) {
			t.Fatalf("with #22 highlighted the Review view leaves out %q:\n%s", kept, view)
		}
	}
	screenIs(t, a, "highlight-review")
}

func TestACommentQueuedWhileHighlightedGoesWhereTheServerSays(t *testing.T) {
	editor, _ := writer(t, "One add is enough.")
	a, _ := opened(t, stack, 100, 30, editor)
	a = press(t, a, "v", "2", "j", "enter")
	a = typed(t, press(t, a, ":"), "7")
	a = press(t, a, "V", "j", "j", "c")
	if a.Bad || a.Diff.Selecting {
		t.Fatalf("the range was not queued: %q", a.Notice)
	}
	if n, ok := presentation.NoteOf(*a.Review, "Sources/Catalog/MemoryCatalog.swift", "queued:4"); !ok || n.Pull != 21 {
		t.Fatalf("queued comment 4 goes on %+v", n)
	}
	filterIs(t, a, "[21]")
}

func TestHighlightedRowsDrawApartFromTheCursorRow(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	for _, dark := range []bool{true, false} {
		th := commonFor(dark).Theme
		s := context.SideContext{Cursor: 1, Rows: []presentation.SideRow{{Key: "a", Text: "#21", Lit: true}, {Key: "b", Text: "#22", Lit: true}, {Key: "c", Text: "#23"}}}
		focused, _ := presentation.SideContent(testCommon().Tr, th, s.Rows, s.Cursor, s.Top, views.Rect{W: 20, H: 3}, true)
		if !strings.HasPrefix(focused[0], th.VisualSelectionBg) || !strings.HasPrefix(focused[1], th.SelectedLineBg) || strings.HasPrefix(focused[2], th.VisualSelectionBg) {
			t.Fatalf("with the keys, a lit row, the cursor and an unlit row draw as %q", focused)
		}
		if th.VisualSelectionBg == th.SelectedLineBg || th.VisualSelectionBg == th.InactiveViewSelectedLineBg {
			t.Fatalf("the highlight's colour is the cursor's: %q", th.VisualSelectionBg)
		}
		away, _ := presentation.SideContent(testCommon().Tr, th, s.Rows, s.Cursor, s.Top, views.Rect{W: 20, H: 3}, false)
		if !strings.HasPrefix(away[1], th.VisualSelectionBg) {
			t.Fatalf("without the keys, the cursor's lit row draws as %q", away[1])
		}
	}
	plain := testCommon().Theme
	plain.Plain = true
	s := context.SideContext{Cursor: 1, Rows: []presentation.SideRow{{Key: "a", Text: "#21", Lit: true}, {Key: "b", Text: "#22", Lit: true}, {Key: "c", Text: "#23"}}}
	focused, _ := presentation.SideContent(testCommon().Tr, plain, s.Rows, s.Cursor, s.Top, views.Rect{W: 20, H: 3}, true)
	away, _ := presentation.SideContent(testCommon().Tr, plain, s.Rows, s.Cursor, s.Top, views.Rect{W: 20, H: 3}, false)
	if !strings.HasPrefix(focused[0], "┃") || !strings.HasPrefix(focused[1], "▌") || !strings.HasPrefix(away[1], "┃") || !strings.HasPrefix(focused[2], " ") {
		t.Fatalf("with no colour, the rows draw as %q, then without the keys %q", focused, away)
	}
}

func TestPathTitlesReadTheSameUnderAHighlight(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	whole := press(t, a, "4", "enter", "enter")
	lit := press(t, a, "j", "v", "4", "enter", "enter")
	title := func(a App) string {
		top := strings.Split(a.View(), "\n")[0]
		return top[strings.Index(top, "[0]"):]
	}
	if title(lit) != title(whole) || !strings.Contains(title(lit), "widgets › Sources─") {
		t.Fatalf("with #22 highlighted the diagram is titled %q, not %q", title(lit), title(whole))
	}
	for _, card := range lit.Diagram.Level.Cards {
		if row := lit.Side(context.DiagramPanel).Rows[lit.Side(context.DiagramPanel).Index(presentation.BoxKey(card.ID))]; row.Title != card.Title {
			t.Fatalf("the Diagram panel titles %s as %q", card.ID, row.Title)
		}
	}
}
