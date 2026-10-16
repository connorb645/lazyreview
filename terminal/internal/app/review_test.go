package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTheReviewViewOfAStack(t *testing.T) {
	a, visited := opened(t, "github.com/octo/widgets/21", 100, 60)
	screenIs(t, a, "review-stack")
	a = press(t, a, "]", "]", "]", "]")
	if got := a.rv.sectionAt(); got != "queue" {
		t.Fatalf("]] twice from the top stopped in %q", got)
	}
	a = press(t, a, "z", "a")
	screenIs(t, a, "review-stack-queue-folded")
	a = press(t, a, "z", "a", "j")
	if it := a.rv.view.items[a.rv.view.cursor]; it.kind != "pull" || it.pull != 21 {
		t.Fatalf("j from the queue's heading stopped on %+v", it)
	}
	a = press(t, a, "j")
	if it := a.rv.view.items[a.rv.view.cursor]; it.kind != "comment" || it.comment != 1 {
		t.Fatalf("j from #21 stopped on %+v", it)
	}
	a = press(t, a, "g", "x")
	if (*visited)[0] != "https://github.com/octo/widgets/pull/21" {
		t.Fatalf("gx on a comment opens its pull request, not %v", *visited)
	}
	a = press(t, a, "g", "X")
	if !strings.HasSuffix((*visited)[1], "/review/github.com/octo/widgets/21#review") {
		t.Fatalf("gX opens the review in the web app, not %s", (*visited)[1])
	}
	a = press(t, a, "G")
	if it := a.rv.view.items[a.rv.view.cursor]; it.kind != "component" {
		t.Fatalf("G stops on the last item, the change's last component, not %+v", it)
	}
}

func TestTheReviewViewOfAPullRequestScrollsToTheCursor(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 80, 14)
	screenIs(t, a, "review-single")
	a = press(t, a, "G")
	screenIs(t, a, "review-single-end")
	bottom := a.rv.view.top
	a = send(t, a, tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})
	if a.rv.view.top >= bottom {
		t.Fatalf("the wheel did not scroll up from %d", bottom)
	}
}
