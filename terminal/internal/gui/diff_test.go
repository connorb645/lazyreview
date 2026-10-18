package gui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func cursorAt(a App) string {
	r := a.Diff.Rows[a.Diff.Cursor]
	side, line, ok := r.At()
	if !ok {
		return r.Path + " " + r.Kind
	}
	return r.Path + " " + side + " " + utils.Itoa(line)
}

func openedOnComment(t *testing.T) App {
	t.Helper()
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "0", "]", "]", "]", "]", "j", "j", "enter")
	if a.Screen() != context.ScreenDiff {
		t.Fatalf("Enter on a queued comment did not open its diff: %q", a.Notice)
	}
	return a
}

func TestEnterOnAQueuedCommentOpensItsLineInTheDiff(t *testing.T) {
	a := openedOnComment(t)
	if got := cursorAt(a); got != "Sources/Catalog/MemoryCatalog.swift RIGHT 10" {
		t.Fatalf("the cursor is on %s", got)
	}
	screenIs(t, a, "diff-stack")
	a = press(t, a, "-")
	if a.Screen() != context.ScreenReview {
		t.Fatal("- went back to the Review view")
	}
	a = press(t, a, "ctrl+o")
	if a.Screen() != context.ScreenDiff || cursorAt(a) != "Sources/Catalog/MemoryCatalog.swift RIGHT 10" {
		t.Fatalf("Ctrl-o went back to the diff, at %v", a.Screen())
	}
	a = press(t, a, "ctrl+o")
	if a.Screen() != context.ScreenReview || a.ReviewView.View.Items[a.ReviewView.View.Cursor].Comment != 1 {
		t.Fatal("Ctrl-o again went back to the comment in the Review view")
	}
	a = press(t, a, "tab")
	if a.Screen() != context.ScreenDiff {
		t.Fatal("Ctrl-i went forward to the diff")
	}
}

func TestMovingThroughTheDiffByLineHunkFileAndSearch(t *testing.T) {
	a := openedOnComment(t)
	a = press(t, a, "g", "g")
	if got := cursorAt(a); got != "Sources/Catalog/MemoryCatalog.swift RIGHT 4" {
		t.Fatalf("gg went to %s, not the first line of the file", got)
	}
	a = press(t, a, "[", "f", "[", "f")
	if got := cursorAt(a); got != "Sources/Basket/Basket.swift file" {
		t.Fatalf("[f [f went to %s", got)
	}
	a = press(t, a, "]", "c")
	if got := cursorAt(a); got != "Sources/Basket/Basket.swift RIGHT 3" {
		t.Fatalf("]c went to the first change of the first hunk, not %s", got)
	}
	a = press(t, a, "]", "f")
	if got := cursorAt(a); got != "Sources/Catalog/MemoryCatalog.swift file" {
		t.Fatalf("]f went to %s", got)
	}
	a = press(t, a, "[", "f")
	if got := cursorAt(a); got != "Sources/Basket/Basket.swift file" {
		t.Fatalf("[f went to %s", got)
	}
	a = press(t, a, "1", "1", "G")
	if got := cursorAt(a); got != "Sources/Basket/Basket.swift RIGHT 11" {
		t.Fatalf("11G went to %s", got)
	}
	a = press(t, a, "/")
	a = typed(t, a, "stored.append")
	if got := cursorAt(a); got != "Sources/Catalog/MemoryCatalog.swift RIGHT 9" {
		t.Fatalf("searching went to %s", got)
	}
	a = press(t, a, ":")
	a = typed(t, a, "2")
	if got := cursorAt(a); got != "Sources/Catalog/MemoryCatalog.swift RIGHT 2" {
		t.Fatalf(":2 went to %s, opening the unchanged lines it sits among", got)
	}
	a = press(t, a, "ctrl+o")
	if got := cursorAt(a); got != "Sources/Catalog/MemoryCatalog.swift RIGHT 9" {
		t.Fatalf("Ctrl-o went back to %s", got)
	}
}

// In the diff, gg and G go to the first and last line of the file at the
// cursor, its header shown above the first, and the Files panel stays on
// that file. In the Files panel, they go to its first and last row.
func TestGgAndGMoveOnlyWhatHasTheKeys(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "2", "j", "j", "j", "j", "enter", "j", "j", "j", "j")
	if a.Focus() != context.MainPanel || cursorAt(a) != "Sources/Catalog/MemoryCatalog.swift RIGHT 5" {
		t.Fatalf("the diff of MemoryCatalog has the keys at %s", cursorAt(a))
	}
	a = press(t, a, "G")
	if got := cursorAt(a); got != "Sources/Catalog/MemoryCatalog.swift RIGHT 11" {
		t.Fatalf("G in the diff went to %s, not the file's last line", got)
	}
	a = press(t, a, "g", "g")
	if got := cursorAt(a); got != "Sources/Catalog/MemoryCatalog.swift RIGHT 4" {
		t.Fatalf("gg in the diff went to %s, not the file's first line", got)
	}
	if r := a.Diff.Rows[a.Diff.Top]; r.Kind != "file" || r.Path != "Sources/Catalog/MemoryCatalog.swift" {
		t.Fatalf("after gg the diff shows from %s %s, not the file's header", r.Path, r.Kind)
	}
	if got := selectedKey(a, context.FilesPanel); got != "file:Sources/Catalog/MemoryCatalog.swift" {
		t.Fatalf("gg and G in the diff moved the Files panel to %s", got)
	}
	a = press(t, a, "esc", "G")
	if got := selectedKey(a, context.FilesPanel); got != "file:Sources/Catalog/Catalog.swift" {
		t.Fatalf("G in the Files panel went to %s, not its last row", got)
	}
	a = press(t, a, "g", "g")
	if got := selectedKey(a, context.FilesPanel); got != "component:root" {
		t.Fatalf("gg in the Files panel went to %s, not its first row", got)
	}
}

func TestFoldsSplitWrapAndTheWholeFile(t *testing.T) {
	a := openedOnComment(t)
	a = press(t, a, "[", "f", "j")
	if a.Diff.Rows[a.Diff.Cursor].Kind != "gap" {
		t.Fatalf("below MemoryCatalog's header is its run of unchanged lines, not %s", a.Diff.Rows[a.Diff.Cursor].Kind)
	}
	a = press(t, a, "z", "o")
	if got := cursorAt(a); !strings.HasPrefix(got, "Sources/Catalog/MemoryCatalog.swift") || a.Diff.Rows[a.Diff.Cursor].Kind != "line" {
		t.Fatalf("zo opened the run, the cursor on its first line, not %s", got)
	}
	a = press(t, a, "z", "c")
	if a.Diff.Rows[a.Diff.Cursor].Kind != "gap" {
		t.Fatal("zc closed the run again")
	}
	a = press(t, a, "z", "M")
	for _, r := range a.Diff.Rows {
		if r.Kind != "file" {
			t.Fatalf("zM closed every file, but a %s row shows", r.Kind)
		}
	}
	a = press(t, a, "z", "R")
	if strings.Count(a.View(), "unchanged line") != 0 {
		t.Fatal("zR opened every run of unchanged lines")
	}
	a = press(t, a, "9", "[", "f", "]", "c", "space", "s")
	screenIs(t, a, "diff-split")
	a = press(t, a, "space", "s", "space", "o")
	screenIs(t, a, "diff-whole")
	a = press(t, a, "space", "o", "9", "[", "f", "enter")
	if !a.Diff.Folded["Sources/Basket/Basket.swift"] {
		t.Fatal("Enter on a file's header folds it")
	}
}

func TestThePullRequestFilterDimsWhatItLeavesOut(t *testing.T) {
	a := openedOnComment(t)
	a = press(t, a, "]", "p")
	if got := fmt.Sprint(a.Filter); got != "[21]" {
		t.Fatalf("]p chose %s", got)
	}
	for _, r := range a.Diff.Rows {
		if r.Path == "Sources/Basket/Basket.swift" {
			t.Fatal("the filter to #21 leaves out a file only #22 changed")
		}
	}
	screenIs(t, a, "diff-filtered")
	a = press(t, a, "[", "p")
	if got := fmt.Sprint(a.Filter); got != "[]" {
		t.Fatalf("[p went back to All, not %s", got)
	}
	a = press(t, a, ":")
	a = typed(t, a, "pull 22")
	if got := fmt.Sprint(a.Filter); got != "[22]" {
		t.Fatalf(":pull 22 chose %s", got)
	}
	a = press(t, a, ":")
	a = typed(t, a, "pull all")
	if got := fmt.Sprint(a.Filter); got != "[]" {
		t.Fatalf(":pull all chose %s", got)
	}
}

func TestAMarkdownFileOfAPullRequest(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 90, 30)
	a = press(t, a, "0", "G", "enter", "space", "w")
	screenIs(t, a, "diff-single")
}

func TestTheWheelReachesTheEndOfAWrappedDiff(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "2", "0", "space", "w")
	main := a.helpers.Window.Layout(a.ContextTree).Main.Inner()
	for i := 0; i < 30; i++ {
		a = send(t, a, tea.MouseMsg{X: main.X + 2, Y: main.Y + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	}
	lines := strings.Split(a.View(), "\n")
	if last := lines[main.Y+main.H-1]; !strings.Contains(last, "4   }") {
		t.Fatalf("the wheel stopped short of the diff's last line; the panel ends with:\n%s", last)
	}
	if got := mainThumb(a); len(got) == 0 || got[len(got)-1] != main.H-1 {
		t.Fatalf("at the end of the diff, the thumb is on rows %v", got)
	}
}
