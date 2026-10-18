package gui

import (
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
)

// In a panel on the left, Ctrl-d, Ctrl-u, Ctrl-f and Ctrl-b move by half
// the panel or the whole of it, a count before them moves that many times,
// and {n}G goes to row n, all in the panel with the keys. A count starts
// with a digit that does not jump to a panel.
func TestPagesAndCountsMoveAPanelOnTheLeft(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "2")
	rows := a.Side(context.FilesPanel).Rows
	height := a.helpers.Window.SideHeight(a.ContextTree, context.FilesPanel)
	half := max(1, height/2)
	steps := []struct {
		keys []string
		row  int
	}{
		{[]string{"ctrl+d"}, min(half, len(rows)-1)},
		{[]string{"ctrl+u"}, 0},
		{[]string{"5", "ctrl+d"}, min(5*half, len(rows)-1)},
		{[]string{"ctrl+b"}, max(0, min(5*half, len(rows)-1)-height)},
		{[]string{"g", "g"}, 0},
		{[]string{"7", "G"}, 6},
		{[]string{"5", "k"}, 1},
		{[]string{"9", "g", "g"}, 8},
	}
	for _, s := range steps {
		a = press(t, a, s.keys...)
		if got, want := selectedKey(a, context.FilesPanel), rows[s.row].Key; a.Focus() != context.FilesPanel || got != want {
			t.Fatalf("%v in the Files panel went to %s in panel %d, not %s", s.keys, got, a.Focus(), want)
		}
	}
}

// In the Review view, Ctrl-d and Ctrl-f move by half a page and a page of
// its lines, and {n}G goes to item n.
func TestPagesAndCountsMoveTheReviewView(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "0")
	height := a.helpers.Window.MainHeight(a.ContextTree)
	want := a.ReviewView.View.Paged(max(1, height/2), height).Cursor
	if a = press(t, a, "ctrl+d"); a.ReviewView.View.Cursor != want || want == 0 {
		t.Fatalf("Ctrl-d in the Review view went to item %d, not %d", a.ReviewView.View.Cursor, want)
	}
	if a = press(t, a, "3", "G"); a.ReviewView.View.Cursor != 2 {
		t.Fatalf("3G in the Review view went to item %d", a.ReviewView.View.Cursor)
	}
}

// A count goes before n and N in the diff and the diagram, and before
// Ctrl-o and Ctrl-i; {n}G in the diagram goes to box n in reading order.
func TestACountGoesBeforeMatchesJumpsAndBoxes(t *testing.T) {
	a := openedOnComment(t)
	a = typed(t, press(t, a, "g", "g", "/"), "catalog")
	if twice, counted := cursorAt(press(t, a, "n", "n")), cursorAt(press(t, a, "2", "n")); counted != twice || counted == cursorAt(press(t, a, "n")) {
		t.Fatalf("2n in the diff went to %s, not %s", counted, twice)
	}
	b := press(t, openedOnComment(t), "-")
	twice, counted := press(t, b, "ctrl+o", "ctrl+o"), press(t, b, "2", "ctrl+o")
	if counted.Screen() != twice.Screen() || counted.ReviewView.View.Cursor != twice.ReviewView.View.Cursor || counted.Screen() != context.ScreenReview {
		t.Fatalf("2 Ctrl-o went to screen %d item %d, not screen %d item %d", counted.Screen(), counted.ReviewView.View.Cursor, twice.Screen(), twice.ReviewView.View.Cursor)
	}
	d := press(t, onDiagram(t), "2", "G")
	focusedOn(t, d, "Basket")
	focusedOn(t, press(t, d, "G", "1", "G"), "MemoryCatalog")
}

// On the home screen, Ctrl-d and Ctrl-u move in the repository's panel
// with the keys.
func TestPagesMoveTheRepositoryWithTheKeys(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	if a = press(t, a, "ctrl+d"); selectedPull(a) != "octo/gadgets#9" {
		t.Fatalf("Ctrl-d went to %s", selectedPull(a))
	}
	if a = press(t, a, "ctrl+u"); selectedPull(a) != "octo/gadgets#7" {
		t.Fatalf("Ctrl-u went to %s", selectedPull(a))
	}
}

// In the Diagram panel, Ctrl-d and Ctrl-u move through the boxes by half
// the panel, and {n}G goes to box n in reading order, or the last; 2 there
// still jumps to the Files panel.
func TestPagesAndCountsMoveTheDiagramPanel(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "4", "enter", "enter", "esc")
	focusedOn(t, a, "sources")
	last := len(a.Diagram.Level.Cards)
	if half := max(1, a.helpers.Window.SideHeight(a.ContextTree, context.DiagramPanel)/2); half < last-1 {
		t.Fatalf("the level has %d boxes, more than half the panel, %d, can show the test", last, half)
	}
	a = press(t, a, "ctrl+d")
	if a.Focus() != context.DiagramPanel || a.Diagram.Focus != last-1 {
		t.Fatalf("Ctrl-d in the Diagram panel went to box %d of %d in panel %d", a.Diagram.Focus+1, last, a.Focus())
	}
	focusedOn(t, press(t, a, "ctrl+u"), "sources")
	if b := press(t, a, "ctrl+u", "5", "G"); b.Focus() != context.DiagramPanel || b.Diagram.Focus != last-1 {
		t.Fatalf("5G in the Diagram panel went to box %d in panel %d", b.Diagram.Focus+1, b.Focus())
	}
	if b := press(t, a, "2"); b.Focus() != context.FilesPanel || b.Keys.Typed() != "" {
		t.Fatalf("2 in the Diagram panel went to panel %d, typed %q", b.Focus(), b.Keys.Typed())
	}
}
