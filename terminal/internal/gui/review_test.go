package gui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTheReviewViewOfAStack(t *testing.T) {
	a, visited := opened(t, "github.com/octo/widgets/21", 100, 60)
	screenIs(t, a, "review-stack")
	a = press(t, a, "0", "]", "]", "]", "]")
	if got := a.ReviewView.SectionAt(); got != "queue" {
		t.Fatalf("]] twice from the top stopped in %q", got)
	}
	a = press(t, a, "z", "a")
	screenIs(t, a, "review-stack-queue-folded")
	a = press(t, a, "z", "a", "j")
	if it := a.ReviewView.View.Items[a.ReviewView.View.Cursor]; it.Kind != "pull" || it.Pull != 21 {
		t.Fatalf("j from the queue's heading stopped on %+v", it)
	}
	a = press(t, a, "j")
	if it := a.ReviewView.View.Items[a.ReviewView.View.Cursor]; it.Kind != "comment" || it.Comment != 1 {
		t.Fatalf("j from #21 stopped on %+v", it)
	}
	a = press(t, a, "g", "x")
	if (*visited)[0] != "https://github.com/octo/widgets/pull/21" {
		t.Fatalf("gx on a comment opens its pull request, not %v", *visited)
	}
	a = press(t, a, "G")
	if it := a.ReviewView.View.Items[a.ReviewView.View.Cursor]; it.Kind != "component" {
		t.Fatalf("G stops on the last item, the change's last component, not %+v", it)
	}
}

func TestTheReviewViewOfAPullRequestScrollsToTheCursor(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 80, 14)
	screenIs(t, a, "review-single")
	a = press(t, a, "0", "G")
	screenIs(t, a, "review-single-end")
	bottom := a.ReviewView.View.Top
	main := a.helpers.Window.Layout(a.ContextTree).Main.Inner()
	a = send(t, a, tea.MouseMsg{X: main.X + 2, Y: main.Y + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})
	if a.ReviewView.View.Top >= bottom {
		t.Fatalf("the wheel did not scroll up from %d", bottom)
	}
}
