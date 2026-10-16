package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func cursorAt(a App) string {
	r := a.diff.rows[a.diff.cursor]
	side, line, ok := r.at()
	if !ok {
		return r.path + " " + r.kind
	}
	return r.path + " " + side + " " + itoa(line)
}

func openedOnComment(t *testing.T) App {
	t.Helper()
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "0", "]", "]", "]", "]", "j", "j", "enter")
	if a.screen != screenDiff {
		t.Fatalf("Enter on a queued comment did not open its diff: %q", a.notice)
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
	if a.screen != screenReview {
		t.Fatal("- went back to the Review view")
	}
	a = press(t, a, "ctrl+o")
	if a.screen != screenDiff || cursorAt(a) != "Sources/Catalog/MemoryCatalog.swift RIGHT 10" {
		t.Fatalf("Ctrl-o went back to the diff, at %v", a.screen)
	}
	a = press(t, a, "ctrl+o")
	if a.screen != screenReview || a.rv.view.items[a.rv.view.cursor].comment != 1 {
		t.Fatal("Ctrl-o again went back to the comment in the Review view")
	}
	a = press(t, a, "tab")
	if a.screen != screenDiff {
		t.Fatal("Ctrl-i went forward to the diff")
	}
}

func TestMovingThroughTheDiffByLineHunkFileAndSearch(t *testing.T) {
	a := openedOnComment(t)
	a = press(t, a, "g", "g")
	if got := cursorAt(a); got != "Sources/Basket/Basket.swift file" {
		t.Fatalf("gg went to %s", got)
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

func TestFoldsSplitWrapAndTheWholeFile(t *testing.T) {
	a := openedOnComment(t)
	a = press(t, a, "[", "f", "j")
	if a.diff.rows[a.diff.cursor].kind != "gap" {
		t.Fatalf("below MemoryCatalog's header is its run of unchanged lines, not %s", a.diff.rows[a.diff.cursor].kind)
	}
	a = press(t, a, "z", "o")
	if got := cursorAt(a); !strings.HasPrefix(got, "Sources/Catalog/MemoryCatalog.swift") || a.diff.rows[a.diff.cursor].kind != "line" {
		t.Fatalf("zo opened the run, the cursor on its first line, not %s", got)
	}
	a = press(t, a, "z", "c")
	if a.diff.rows[a.diff.cursor].kind != "gap" {
		t.Fatal("zc closed the run again")
	}
	a = press(t, a, "z", "M")
	for _, r := range a.diff.rows {
		if r.kind != "file" {
			t.Fatalf("zM closed every file, but a %s row shows", r.kind)
		}
	}
	a = press(t, a, "z", "R")
	if strings.Count(a.View(), "unchanged line") != 0 {
		t.Fatal("zR opened every run of unchanged lines")
	}
	a = press(t, a, "g", "g", "]", "c", "space", "s")
	screenIs(t, a, "diff-split")
	a = press(t, a, "space", "s", "space", "o")
	screenIs(t, a, "diff-whole")
	a = press(t, a, "space", "o", "g", "g", "enter")
	if !a.diff.folded["Sources/Basket/Basket.swift"] {
		t.Fatal("Enter on a file's header folds it")
	}
}

func TestThePullRequestFilterDimsWhatItLeavesOut(t *testing.T) {
	a := openedOnComment(t)
	a = press(t, a, "]", "p")
	if a.filter != 21 {
		t.Fatalf("]p chose %d", a.filter)
	}
	for _, r := range a.diff.rows {
		if r.path == "Sources/Basket/Basket.swift" {
			t.Fatal("the filter to #21 leaves out a file only #22 changed")
		}
	}
	screenIs(t, a, "diff-filtered")
	a = press(t, a, "[", "p")
	if a.filter != 0 {
		t.Fatalf("[p went back to All, not %d", a.filter)
	}
	a = press(t, a, ":")
	a = typed(t, a, "pull 22")
	if a.filter != 22 {
		t.Fatalf(":pull 22 chose %d", a.filter)
	}
	a = press(t, a, ":")
	a = typed(t, a, "pull all")
	if a.filter != 0 {
		t.Fatalf(":pull all chose %d", a.filter)
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
	main := a.layout().main.inner()
	for i := 0; i < 30; i++ {
		a = send(t, a, tea.MouseMsg{X: main.x + 2, Y: main.y + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	}
	lines := strings.Split(a.View(), "\n")
	if last := lines[main.y+main.h-1]; !strings.Contains(last, "4   }") {
		t.Fatalf("the wheel stopped short of the diff's last line; the panel ends with:\n%s", last)
	}
	if got := mainThumb(a); len(got) == 0 || got[len(got)-1] != main.h-1 {
		t.Fatalf("at the end of the diff, the thumb is on rows %v", got)
	}
}
