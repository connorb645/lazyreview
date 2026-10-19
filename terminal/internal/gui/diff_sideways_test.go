package gui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

// onSearch is the stack at 64 by 24 with the diff of Search.swift, whose
// filter line runs past the main panel, and the keys in it.
func onSearch(t *testing.T) App {
	t.Helper()
	a, _ := opened(t, stack, 64, 24)
	a = press(t, a, "2", "5", "j", "0")
	if got := cursorAt(a); got != search+" file" {
		t.Fatalf("the diff of Search.swift has the keys, at %s", got)
	}
	return a
}

func TestHAndLScrollTheCodeAndStopAtItsEnds(t *testing.T) {
	a := onSearch(t)
	height := a.helpers.Window.MainHeight(a.ContextTree)
	step, end := presentation.ScrollStep(a.Diff.DiffView), presentation.ScrollEnd(a.Diff.DiffView, height)
	if end <= step {
		t.Fatalf("the longest line runs %d columns past the panel, too few to test with a step of %d", end, step)
	}
	if a = press(t, a, "l"); a.Diff.Left != step {
		t.Fatalf("l scrolls %d columns, not %d", step, a.Diff.Left)
	}
	screenIs(t, a, "diff-scrolled")
	if a = press(t, a, "2", "0", "l"); a.Diff.Left != end {
		t.Fatalf("20l stops where the longest line ends at the edge, %d, not %d", end, a.Diff.Left)
	}
	if a = press(t, a, "j", "j", "k"); a.Diff.Left != end {
		t.Fatalf("j and k keep the scroll, not %d", a.Diff.Left)
	}
	if a = press(t, a, "h"); a.Diff.Left != end-step {
		t.Fatalf("h scrolls back a step, to %d", a.Diff.Left)
	}
	if a = press(t, a, "9", "h"); a.Diff.Left != 0 {
		t.Fatalf("h stops at the first column, not %d", a.Diff.Left)
	}
}

func TestSplitSidesScrollTogetherAndWrapOrRenderedDoNot(t *testing.T) {
	a := press(t, onSearch(t), "space", "s", "l", "l")
	if a.Diff.Left == 0 {
		t.Fatal("l scrolls a split diff")
	}
	screenIs(t, a, "diff-scrolled-split")
	wrapped := press(t, a, "space", "w")
	if left := wrapped.Diff.Left; press(t, wrapped, "l").Diff.Left != left || press(t, wrapped, "h").Diff.Left != left {
		t.Fatal("h and l do nothing while lines wrap")
	}
	b, _ := opened(t, "github.com/octo/widgets/30", 60, 30)
	b = press(t, b, "2", "j", "0", "space", "r", "l")
	if !b.Diff.Rendered[b.Diff.File()] || b.Diff.Left != 0 {
		t.Fatalf("l does nothing in a rendered Markdown file: rendered %v, scrolled %d", b.Diff.Rendered, b.Diff.Left)
	}
}

func TestTheScrollGoesBackWhenTheDiffShowsOtherFiles(t *testing.T) {
	a := press(t, onSearch(t), "l", "esc", "k")
	if a.Diff.Left != 0 {
		t.Fatalf("another file's diff starts at the first column, not %d", a.Diff.Left)
	}
}

func TestTheSidewaysWheelAndShiftWithTheWheelScrollTheCode(t *testing.T) {
	a := onSearch(t)
	main := a.helpers.Window.Layout(a.ContextTree).Main.Inner()
	wheel := func(a App, button tea.MouseButton, shift bool) App {
		return send(t, a, tea.MouseMsg{X: main.X + 2, Y: main.Y + 2, Action: tea.MouseActionPress, Button: button, Shift: shift})
	}
	if a = wheel(a, tea.MouseButtonWheelRight, false); a.Diff.Left != 6 {
		t.Fatalf("the wheel to the right scrolls 6 columns, not %d", a.Diff.Left)
	}
	if a = wheel(a, tea.MouseButtonWheelDown, true); a.Diff.Left != 12 {
		t.Fatalf("Shift and the wheel down scroll 6 more, to %d", a.Diff.Left)
	}
	if a = wheel(wheel(a, tea.MouseButtonWheelLeft, false), tea.MouseButtonWheelUp, true); a.Diff.Left != 0 {
		t.Fatalf("the wheel to the left and Shift and the wheel up scroll back, to %d", a.Diff.Left)
	}
}

func TestTheScrollKeysComeFromTheConfig(t *testing.T) {
	cmn := handedOver(t, map[string]map[string][]string{"diff": {"scrollRight": {"L"}, "scrollLeft": {}}})
	a, _ := openedIn(t, cmn, &changes{answers: map[string]answer{}}, stack, 64, 24)
	a = press(t, a, "2", "5", "j", "0", "l")
	if a.Diff.Left != 0 {
		t.Fatal("l no longer scrolls once the config gives scrollRight another key")
	}
	if a = press(t, a, "L"); a.Diff.Left == 0 || press(t, a, "h").Diff.Left != a.Diff.Left {
		t.Fatalf("L scrolls, and h, disabled, does not: %d", a.Diff.Left)
	}
	if !strings.Contains(a.View(), "1 + …") {
		t.Fatal("the scrolled diff marks the text hidden on the left")
	}
}
