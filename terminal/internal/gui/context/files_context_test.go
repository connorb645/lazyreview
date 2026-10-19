package context

import (
	"slices"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

// filesPanel is a Files panel of made-up rows: a component of two folders,
// then a component of one file, the cursor on a row.
func filesPanel(cursor int) FilesContext {
	var rows []presentation.SideRow
	for _, key := range []string{"component:ovens", "dir:ovens:heat", "file:heat/burner.ts", "file:heat/grill.ts", "dir:ovens:timer", "file:timer/alarm.ts", "component:orders", "file:queue.ts"} {
		rows = append(rows, presentation.SideRow{Key: key})
	}
	return FilesContext{SideContext: SideContext{Rows: rows, Cursor: cursor}}
}

func moved(f FilesContext, by int) FilesContext {
	f.SideContext = f.Move(by, 20)
	return f.FromCursor()
}

func TestTheCursorChoosesItsRowAlone(t *testing.T) {
	f := moved(filesPanel(0), 2)
	if !slices.Equal(f.Chosen, []string{"file:heat/burner.ts"}) || f.HasSelection() {
		t.Fatalf("a move chose %v", f.Chosen)
	}
}

func TestASelectionGrowsAndShrinksFromItsFirstRow(t *testing.T) {
	f := filesPanel(2).SelectionToggled()
	if f.GrowFrom != "file:heat/burner.ts" || !slices.Equal(f.Chosen, []string{"file:heat/burner.ts"}) {
		t.Fatalf("v starts on the cursor's row: from %q, %v", f.GrowFrom, f.Chosen)
	}
	for _, step := range []struct {
		by   int
		want []string
	}{
		{3, []string{"file:heat/burner.ts", "file:heat/grill.ts", "dir:ovens:timer", "file:timer/alarm.ts"}},
		{-2, []string{"file:heat/burner.ts", "file:heat/grill.ts"}},
		{-1, []string{"file:heat/burner.ts"}},
		{-2, []string{"component:ovens", "dir:ovens:heat", "file:heat/burner.ts"}},
		{9, []string{"file:heat/burner.ts", "file:heat/grill.ts", "dir:ovens:timer", "file:timer/alarm.ts", "component:orders", "file:queue.ts"}},
	} {
		if f = moved(f, step.by); !slices.Equal(f.Chosen, step.want) || !f.HasSelection() {
			t.Fatalf("moving by %d chose %v, not %v", step.by, f.Chosen, step.want)
		}
	}
}

func TestVOrStoppingEndsTheSelectionOnTheCursorsRow(t *testing.T) {
	growing := moved(filesPanel(2).SelectionToggled(), 2)
	for name, f := range map[string]FilesContext{"v": growing.SelectionToggled(), "Esc": growing.SelectionStopped()} {
		if f.GrowFrom != "" || !slices.Equal(f.Chosen, []string{"dir:ovens:timer"}) || f.HasSelection() {
			t.Fatalf("%s ended the selection: from %q, %v", name, f.GrowFrom, f.Chosen)
		}
		if f = moved(f, 1); !slices.Equal(f.Chosen, []string{"file:timer/alarm.ts"}) {
			t.Fatalf("after %s a move chooses its row alone, not %v", name, f.Chosen)
		}
	}
}

func TestASelectionHeldAfterItStopsGrowingIsASelection(t *testing.T) {
	f := moved(filesPanel(2).SelectionToggled(), 1)
	f.GrowFrom = ""
	if !f.HasSelection() {
		t.Fatal("two rows chosen, no longer growing, are still a selection")
	}
	f.Chosen = []string{"component:ovens"}
	if !f.HasSelection() {
		t.Fatal("a row chosen that the cursor is not on is a selection")
	}
}

func TestASelectionWhoseFirstRowIsGoneStopsGrowing(t *testing.T) {
	f := moved(filesPanel(2).SelectionToggled(), 1)
	f.Rows = slices.Delete(slices.Clone(f.Rows), 2, 3)
	f.Cursor = 2
	if f = f.FromCursor(); f.GrowFrom != "" || !slices.Equal(f.Chosen, []string{"file:heat/grill.ts"}) {
		t.Fatalf("with its first row gone, the selection chose %v from %q", f.Chosen, f.GrowFrom)
	}
}
