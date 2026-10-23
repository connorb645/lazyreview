package gui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTheDiffOfAReviewThatTakesNoCommentsOffersNoCommentKey(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	a = press(t, a, "2", "j", "j", "enter")
	if bar := lastLine(a); !strings.Contains(bar, "Comment: c") {
		t.Fatalf("a pull request's diff does not offer to comment: %q", bar)
	}
	state := a.Review.State
	state.Kind, state.Members, state.Queue, state.GitHub = "branch", nil, nil, nil
	a = relaid(a.with(a.helpers.Refresh.WithState(a.ContextTree, state)))
	if bar := lastLine(a); strings.Contains(bar, "Comment") || !strings.Contains(bar, "Ask Claude: a") {
		t.Errorf("a branch's diff offers %q", bar)
	}
	a = press(t, a, "j", "j", "V")
	if bar := lastLine(a); strings.Contains(bar, "Comment") || !strings.Contains(bar, "Ask Claude: a") {
		t.Errorf("a selection in a branch's diff offers %q", bar)
	}
}

func lastLine(a App) string {
	lines := strings.Split(ansi.Strip(a.View()), "\n")
	return lines[len(lines)-1]
}
