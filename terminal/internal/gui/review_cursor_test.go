package gui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/models"
)

func litKeys(a App) string {
	var lit []string
	for _, r := range a.Side(context.ReviewPanel).Rows {
		if r.Lit {
			lit = append(lit, r.Key)
		}
	}
	return strings.Join(lit, " ")
}

func TestAStackOpensWithTheReviewPanelsCursorOnAll(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	if got := strings.Join(rowKeys(a, context.ReviewPanel), " "); got != "all pull:21 pull:22" {
		t.Fatalf("a stack's Review panel lists %s", got)
	}
	if got := selectedKey(a, context.ReviewPanel); got != "all" {
		t.Fatalf("a stack opens with the cursor on %s", got)
	}
	filterIs(t, a, "[]")
	if !strings.Contains(a.View(), "╭─[1]─Review──") {
		t.Fatalf("the Review panel's title still holds the stack's tabs:\n%s", a.View())
	}
}

func TestTheReviewPanelsCursorFiltersTheStack(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "j")
	filterIs(t, a, "[21]")
	for _, key := range rowKeys(a, context.FilesPanel) {
		if key == "file:Sources/Basket/Basket.swift" || key == "file:web/src/cart.ts" {
			t.Fatalf("with the cursor on #21 the Files panel shows %s, which only #22 changed", key)
		}
	}
	if got := strings.Join(rowKeys(a, context.NotesPanel), " "); got != "queued:1 thread:T1" {
		t.Fatalf("with the cursor on #21 the Notes panel shows %s", got)
	}
	if !strings.Contains(a.View(), "Showing #21 only.") {
		t.Fatalf("with the cursor on #21 the Review view shows the whole stack:\n%s", a.View())
	}
	screenIs(t, a, "cursor-on-one")
	a = press(t, a, "j")
	filterIs(t, a, "[22]")
	for _, key := range rowKeys(a, context.FilesPanel) {
		if key == "file:Sources/Catalog/Search.swift" {
			t.Fatalf("with the cursor on #22 the Files panel shows %s, which only #21 changed", key)
		}
	}
	a = press(t, a, "k", "k")
	filterIs(t, a, "[]")
	if got := selectedKey(a, context.ReviewPanel); got != "all" {
		t.Fatalf("k from the first pull request went to %s, not All", got)
	}
	a = press(t, a, "G")
	filterIs(t, a, "[22]")
	a = press(t, a, "g", "g")
	filterIs(t, a, "[]")
}

func TestTheDiagramFollowsTheReviewPanelsCursor(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "j", "4", "enter", "enter", "enter")
	filterIs(t, a, "[21]")
	dimmed := map[string]bool{}
	for _, card := range a.Diagram.Level.Cards {
		dimmed[card.ID] = presentation.Dimmed(card, a.Filter)
	}
	if !dimmed["Basket"] || dimmed["Search"] {
		t.Fatalf("with the cursor on #21 the diagram dims %v", dimmed)
	}
}

func TestAClickOnAReviewPanelRowFiltersTheStack(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "2")
	review := a.helpers.Window.Layout(a.ContextTree).Sides[context.ReviewPanel].Inner()
	a = send(t, a, tea.MouseMsg{X: review.X + 4, Y: review.Y + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if a.Focus() != context.ReviewPanel || selectedKey(a, context.ReviewPanel) != "pull:22" {
		t.Fatalf("a click chose panel %d, %s", a.Focus(), selectedKey(a, context.ReviewPanel))
	}
	filterIs(t, a, "[22]")
	a = send(t, a, tea.MouseMsg{X: review.X + 4, Y: review.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	filterIs(t, a, "[]")
}

func TestTheCursorsFilterStaysWhenTheKeysLeaveAndComeBack(t *testing.T) {
	for _, keys := range [][]string{{"2"}, {"3"}, {"4"}, {"h"}, {"l"}, {"0"}, {"enter"}} {
		a, _ := opened(t, stack, 100, 30)
		a = press(t, a, "j")
		a = press(t, a, keys...)
		a = press(t, a, "j", "k")
		filterIs(t, a, "[21]")
		a = press(t, a, "1")
		filterIs(t, a, "[21]")
		if got := selectedKey(a, context.ReviewPanel); got != "pull:21" {
			t.Fatalf("after %v and back, the cursor is on %s", keys, got)
		}
	}
}

func TestTheReviewViewLeavesTheReviewPanelsCursorWhereItIs(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "0")
	for i := 0; ; i++ {
		if v := a.ReviewView.View; v.Cursor < len(v.Items) && v.Items[v.Cursor].Pull == 22 {
			break
		}
		if i == 30 {
			t.Fatal("j in the Review view never reached #22")
		}
		a = press(t, a, "j")
	}
	if got := selectedKey(a, context.ReviewPanel); got != "all" {
		t.Fatalf("the Review view moved the Review panel's cursor to %s", got)
	}
	filterIs(t, a, "[]")
}

func TestEscInTheReviewPanelPutsTheCursorOnAll(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "j", "j", "esc")
	filterIs(t, a, "[]")
	if got := selectedKey(a, context.ReviewPanel); got != "all" || a.Notice != "" {
		t.Fatalf("Esc put the cursor on %s and said %q", got, a.Notice)
	}
	a = press(t, a, "j", "v", "j", "esc")
	filterIs(t, a, "[]")
	if got := selectedKey(a, context.ReviewPanel); got != "all" || a.GrowFrom != 0 {
		t.Fatalf("Esc on a highlight put the cursor on %s, growing from %d", got, a.GrowFrom)
	}
	a = press(t, a, "esc")
	if a.Notice != "ZZ or :q quits" {
		t.Fatalf("Esc on All says %q", a.Notice)
	}
}

func TestBracketPMovesTheReviewPanelsCursorWithTheFilter(t *testing.T) {
	a := onThreeStack(t)
	for _, want := range []string{"pull:21", "pull:22", "pull:23", "all"} {
		a = press(t, a, "]", "p")
		if got := selectedKey(a, context.ReviewPanel); got != want {
			t.Fatalf("]p put the cursor on %s, not %s", got, want)
		}
	}
	filterIs(t, a, "[]")
	a = press(t, a, "[", "p")
	filterIs(t, a, "[23]")
	a = press(t, a, "2", "[", "p")
	filterIs(t, a, "[22]")
	if got := selectedKey(a, context.ReviewPanel); got != "pull:22" {
		t.Fatalf("[p from the Files panel left the Review panel's cursor on %s", got)
	}
	a = press(t, a, "1")
	filterIs(t, a, "[22]")
}

func TestPullCommandMovesTheReviewPanelsCursor(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, typed(t, press(t, a, ":"), "pull 22"), "enter")
	filterIs(t, a, "[22]")
	if got := selectedKey(a, context.ReviewPanel); got != "pull:22" {
		t.Fatalf(":pull 22 left the cursor on %s", got)
	}
	a = press(t, typed(t, press(t, a, ":"), "pull all"), "enter")
	if got := selectedKey(a, context.ReviewPanel); got != "all" {
		t.Fatalf(":pull all left the cursor on %s", got)
	}
}

func TestVOnAllDoesNothingAndAHighlightGrowsOnlyOverPullRequests(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "v")
	filterIs(t, a, "[]")
	if a.GrowFrom != 0 || strings.Contains(a.View(), "VISUAL ·") {
		t.Fatal("v on All started a highlight")
	}
	a = press(t, a, "G", "v", "k", "k")
	filterIs(t, a, "[21 22]")
	if got := selectedKey(a, context.ReviewPanel); got != "pull:21" {
		t.Fatalf("a growing highlight took the cursor to %s", got)
	}
	a = press(t, a, "g", "g")
	filterIs(t, a, "[21 22]")
	if got := selectedKey(a, context.ReviewPanel); got != "pull:21" {
		t.Fatalf("gg in a growing highlight took the cursor to %s", got)
	}
}

func TestTheFiltersRowsAreLitSoAPanelWithoutTheKeysShowsThem(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	if got := litKeys(a); got != "all" {
		t.Fatalf("on All, %q is lit", got)
	}
	a = press(t, a, "j", "j", "2")
	if got := litKeys(a); got != "pull:22" {
		t.Fatalf("with the cursor on #22, %q is lit", got)
	}
	if !strings.Contains(a.View(), "│┃#22 Add a basket") {
		t.Fatalf("with no colour, the Review panel without the keys does not mark #22:\n%s", a.View())
	}
	a = press(t, a, "1", "k", "v", "j")
	if got := litKeys(a); got != "all pull:21 pull:22" {
		t.Fatalf("a highlight of both, the whole stack, lights %q", got)
	}
}

func TestTheBarOffersOnAllOnlyWhatWorksThere(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	words := func(a App) string {
		var out []string
		for _, h := range a.controllers.Hints(a.ContextTree) {
			out = append(out, h[1]+": "+h[0])
		}
		return strings.Join(out, " | ")
	}
	if got := words(a); got != "Review: Enter | Panels: h l | Keys: Space ?" {
		t.Fatalf("on All the bar says %q", got)
	}
	if got := words(press(t, a, "j")); got != "Review: Enter | Highlight: v | All: Esc | Post: P | GitHub: gx | Panels: h l | Keys: Space ?" {
		t.Fatalf("on a pull request the bar says %q", got)
	}
}

func TestTheBarOffersPostingAndGitHubOnlyWhereTheyWork(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	words := func(a App) string {
		var out []string
		for _, h := range a.controllers.Hints(a.ContextTree) {
			out = append(out, h[1]+": "+h[0])
		}
		return strings.Join(out, " | ")
	}
	if got := words(a); got != "Review: Enter | Post: P | GitHub: gx | Panels: h l | Keys: Space ?" {
		t.Fatalf("on a pull request's review the bar says %q", got)
	}
	s := a.ContextTree
	moved, branch := *s.Review, *s.Review
	queue := *moved.State.Queue
	queue.Moved = &models.Moved{To: stack, Stack: []int{21, 30}}
	moved.State.Queue = &queue
	s.Review = &moved
	if got := words(a.with(s)); got != "Review: Enter | GitHub: gx | Panels: h l | Keys: Space ?" {
		t.Fatalf("on a review that moved the bar says %q", got)
	}
	branch.State.Kind, branch.State.Members, branch.State.Queue = "branch", nil, nil
	s.Review = &branch
	if got := words(a.with(s)); got != "Review: Enter | Panels: h l | Keys: Space ?" {
		t.Fatalf("on a branch's review the bar says %q", got)
	}
	if got := words(press(t, a.with(s), "3")); got != "Open: Enter | Panels: h l | Keys: Space ?" {
		t.Fatalf("on a branch's review the Notes panel's bar says %q", got)
	}
}

func TestTheReviewPanelsCursorIsKeptAcrossAReload(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "j", "j", "2")
	a = send(t, a, types.ReviewReloaded{Data: *a.Review})
	filterIs(t, a, "[22]")
	if got := selectedKey(a, context.ReviewPanel); got != "pull:22" {
		t.Fatalf("a reload moved the cursor to %s", got)
	}
	d := *a.Review
	stacked := *d.Page.Stack
	stacked.Members = stacked.Members[:1]
	d.Page.Stack = &stacked
	a = send(t, a, types.ReviewReloaded{Data: d})
	filterIs(t, a, "[]")
	if got := selectedKey(a, context.ReviewPanel); got != "all" {
		t.Fatalf("with #22 gone from the stack the cursor is on %s", got)
	}
}

func TestAReviewOfOnePullRequestHasNoAllRow(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	if got := strings.Join(rowKeys(a, context.ReviewPanel), " "); got != "pull:30" {
		t.Fatalf("a review of one pull request lists %s", got)
	}
	a = press(t, a, "j", "k")
	filterIs(t, a, "[]")
}

func TestTheReviewPanelShowsAllAndThreePullRequestsAt80By24(t *testing.T) {
	a := send(t, onThreeStack(t), tea.WindowSizeMsg{Width: 80, Height: 24})
	if h := a.helpers.Window.SideHeight(a.ContextTree, context.ReviewPanel); h != 4 {
		t.Fatalf("at 80 by 24 the Review panel shows %d of its 4 rows", h)
	}
}
