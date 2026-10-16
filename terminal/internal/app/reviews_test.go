package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTheListOfReviewsOpensAReviewAndFindsOne(t *testing.T) {
	a, visited := opened(t, "", 100, 16)
	screenIs(t, a, "reviews")
	a = press(t, a, "j")
	if a.reviews.view.cursor != 1 {
		t.Fatalf("j moved to %d", a.reviews.view.cursor)
	}
	a = press(t, a, "g", "g")
	if a.reviews.view.cursor != 0 {
		t.Fatalf("gg moved to %d", a.reviews.view.cursor)
	}
	a = press(t, a, "/")
	a = typed(t, a, "basket")
	if got := a.reviews.reviews[a.reviews.view.cursor].ID; got != "github.com/octo/widgets/21" {
		t.Fatalf("finding basket stopped on %s", got)
	}
	a = press(t, a, "g", "x")
	if len(*visited) != 1 || (*visited)[0] != "https://github.com/octo/widgets/pull/21" {
		t.Fatalf("gx opened %v", *visited)
	}
	a = press(t, a, "enter")
	if a.screen != screenReview || a.review == nil || a.review.id != "github.com/octo/widgets/21" {
		t.Fatalf("Enter did not open the review: screen %d", a.screen)
	}
	a = press(t, a, "g", "h")
	if a.screen != screenReviews {
		t.Fatal("gh did not go back to every review")
	}
	a = press(t, a, "space", "v")
	if a.screen != screenReview {
		t.Fatal("Space v did not go back to the review open")
	}
}

func TestTheKeysOfAViewAndTheCommandLine(t *testing.T) {
	a, _ := opened(t, "", 100, 24)
	a = press(t, a, "space")
	if !strings.Contains(a.View(), "Space ? keys") {
		t.Fatalf("after the leader, the hint line names the keys that can follow:\n%s", a.View())
	}
	a = press(t, a, "?")
	screenIs(t, a, "reviews-keys")
	a = press(t, a, "q")
	if a.panel != nil {
		t.Fatal("q did not close the keys")
	}
	a = press(t, a, ":")
	a = typed(t, a, "frob")
	if a.notice != "not a command: :frob" || !a.bad {
		t.Fatalf("an unknown command says so, not %q", a.notice)
	}
	a = press(t, a, ":")
	for _, r := range "q" {
		a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if _, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil || cmd() != tea.Quit() {
		t.Fatal(":q did not quit")
	}
	if _, cmd := press(t, a, "esc").Update(keyMsg("Z")); cmd != nil {
		t.Fatal("Z alone quits nothing")
	}
}

func TestAClickPicksAReviewAndADoubleClickOpensIt(t *testing.T) {
	a, _ := opened(t, "", 100, 16)
	a = send(t, a, tea.MouseMsg{X: 4, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if a.reviews.view.cursor != 1 {
		t.Fatalf("a click on the second review picked %d", a.reviews.view.cursor)
	}
	a = send(t, a, tea.MouseMsg{X: 4, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if a.screen != screenReview || a.review == nil || a.review.id != "github.com/octo/widgets/21" {
		t.Fatal("a double click did not open the review")
	}
}
