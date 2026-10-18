package gui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/common"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func numberedAs(t *testing.T, numbers string) *common.Common {
	t.Helper()
	var handOver map[string]any
	if err := json.Unmarshal([]byte(defaultHandOver()), &handOver); err != nil {
		t.Fatal(err)
	}
	handOver["lineNumbers"] = numbers
	text, err := json.Marshal(handOver)
	if err != nil {
		t.Fatal(err)
	}
	return commonFrom(string(text), true)
}

func onBasketNumbered(t *testing.T, numbers string) App {
	t.Helper()
	a, _ := openedIn(t, numberedAs(t, numbers), &changes{answers: map[string]answer{}}, stack, 100, 30)
	return typed(t, press(t, a, "space", "f"), "Basket/Basket")
}

// gutters are the numbers each row of the diff shows, by row, as drawn:
// one string for a unified row, the left's and the right's for a split one.
func gutters(a App) map[int][]string {
	lines := presentation.DiffLines(a.c.Tr, a.c.Theme, a.controllers.Labels(context.DiffKey), *a.Review, a.Diff.DiffView, a.helpers.Window.MainHeight(a.ContextTree), a.ShownPulls(), true)
	out := map[int][]string{}
	for i, line := range lines {
		row := a.Diff.Top + i
		r, cells := a.Diff.Rows[row], []rune(line)
		switch {
		case r.Kind != "line":
		case r.Line != nil:
			out[row] = []string{strings.Join(strings.Fields(string(cells[2:12])), " ")}
		default:
			half := (a.Diff.Width - 3) / 2
			out[row] = []string{strings.TrimSpace(string(cells[2:6])), strings.TrimSpace(string(cells[half+3 : half+7]))}
		}
	}
	return out
}

// With relative numbers, the cursor's line shows its own numbers, and
// every other line how many rows it is from the cursor, the count j or k
// takes to reach it, in the unified view and the split one alike. A side
// of a split row with no line shows no number.
func TestRelativeNumbersAreTheCountToEachLine(t *testing.T) {
	a := press(t, onBasketNumbered(t, "relative"), "]", "c", "j")
	screenIs(t, a, "diff-relative")
	for _, split := range []bool{false, true} {
		if split {
			a = press(t, a, "space", "s")
			screenIs(t, a, "diff-relative-split")
		}
		for row, shown := range gutters(a) {
			r, away := a.Diff.Rows[row], max(row-a.Diff.Cursor, a.Diff.Cursor-row)
			for side, got := range shown {
				want := utils.Itoa(away)
				switch {
				case split && (side == 0 && r.Left == nil || side == 1 && r.Right == nil):
					want = ""
				case row == a.Diff.Cursor && !split:
					_, line, _ := r.At()
					if !strings.Contains(got, utils.Itoa(line)) {
						t.Fatalf("the cursor's line %d shows %q", line, got)
					}
					continue
				case row == a.Diff.Cursor && side == 0:
					want = utils.Itoa(r.Left.Old)
				case row == a.Diff.Cursor:
					want = utils.Itoa(r.Right.New)
				}
				if got != want {
					t.Fatalf("row %d, %d rows from the cursor, shows %q on side %d", row, away, got, side)
				}
			}
		}
	}
	below := a.Diff.Cursor + 6
	b := press(t, a, "6", "j")
	if b.Diff.Cursor != below || gutters(a)[below][0] != "6" {
		t.Fatalf("6j went to row %d, not row %d, which shows %q", b.Diff.Cursor, below, gutters(a)[below])
	}
}

// With no numbers, a line's text starts right after its sign, in the
// unified view and on each side of the split one.
func TestNoNumbersGiveTheCodeTheirRoom(t *testing.T) {
	a := press(t, onBasketNumbered(t, "none"), "]", "c", "j")
	screenIs(t, a, "diff-none")
	for row, shown := range gutters(a) {
		if strings.ContainsAny(shown[0], "0123456789") {
			t.Fatalf("row %d shows %q before its text", row, shown[0])
		}
	}
	screenIs(t, press(t, a, "space", "s"), "diff-none-split")
}
