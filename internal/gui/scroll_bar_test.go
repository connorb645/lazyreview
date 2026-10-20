package gui

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/gui/views"
)

func thumbIn(lines []string, x int) []int {
	var rows []int
	for y, line := range lines {
		if ansi.TruncateLeft(ansi.Truncate(line, x+1, ""), x, "") == "▐" {
			rows = append(rows, y)
		}
	}
	return rows
}

func mainThumb(a App) []int {
	r := a.helpers.Window.Layout(a.ContextTree).Main
	return thumbIn(strings.Split(a.View(), "\n")[r.Y+1:r.Y+r.H-1], r.X+r.W-1)
}

func through(first, last int) []int {
	var rows []int
	for y := first; y <= last; y++ {
		rows = append(rows, y)
	}
	return rows
}

func TestTheThumbIsSizedAndPlacedAsLazygitDoes(t *testing.T) {
	for _, c := range []struct {
		name       string
		lines, top int
		want       []int
	}{
		{"content that fits shows no bar", 21, 0, nil},
		{"two pages, at the top", 42, 0, through(0, 10)},
		{"two pages, one line down", 42, 1, through(1, 11)},
		{"two pages, one line from the end", 42, 20, through(9, 19)},
		{"two pages, at the end", 42, 21, through(10, 20)},
		{"two pages, past the end", 42, 30, through(10, 20)},
		{"a long file keeps a thumb of one row", 1000, 490, through(10, 10)},
	} {
		box := views.Framed(nil, views.Extent{Lines: c.lines, Top: c.top}, 10, 23, "", "", "", false, testCommon().Theme)
		if got := thumbIn(box[1:len(box)-1], 9); !slices.Equal(got, c.want) {
			t.Errorf("%s: the thumb is on rows %v, not %v", c.name, got, c.want)
		}
	}
}

func TestTheDiffShowsWhereItIsInTheMainPanelsBorder(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "2", "0")
	last := a.helpers.Window.MainHeight(a.ContextTree) - 1
	top := mainThumb(a)
	if len(top) == 0 || top[0] != 0 || top[len(top)-1] == last {
		t.Fatalf("at the top of the diff, the thumb is on rows %v", top)
	}
	a = press(t, a, "ctrl+d")
	middle := mainThumb(a)
	if len(middle) != len(top) || middle[0] == 0 || middle[len(middle)-1] == last {
		t.Fatalf("halfway down the diff, the thumb is on rows %v", middle)
	}
	a = press(t, a, "9", "]", "f", "G")
	end := mainThumb(a)
	if len(end) != len(top) || end[len(end)-1] != last {
		t.Fatalf("at the end of the diff, the thumb is on rows %v", end)
	}
}

func TestADiffThatFitsShowsNoBar(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "2", "G")
	if got := cursorAt(a); got != "Sources/Catalog/Catalog.swift file" {
		t.Fatalf("G in the Files panel chose %s", got)
	}
	if got := mainThumb(a); got != nil {
		t.Fatalf("a diff that fits shows a thumb on rows %v", got)
	}
}

func TestTheRenderedMarkdownShowsTheSameBar(t *testing.T) {
	a := send(t, onGuide(t), tea.WindowSizeMsg{Width: 100, Height: 12})
	if got := mainThumb(a); len(got) == 0 || got[0] != 0 {
		t.Fatalf("at the top of the rendered file, the thumb is on rows %v", got)
	}
	a = press(t, a, "G")
	if got := mainThumb(a); len(got) == 0 || got[len(got)-1] != a.helpers.Window.MainHeight(a.ContextTree)-1 {
		t.Fatalf("at the end of the rendered file, the thumb is on rows %v", got)
	}
}

func TestTheReviewViewShowsTheSameBar(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	if got := mainThumb(a); len(got) == 0 || got[0] != 0 {
		t.Fatalf("at the top of the Review view, the thumb is on rows %v", got)
	}
	a = press(t, a, "0", "G")
	if got := mainThumb(a); len(got) == 0 || got[len(got)-1] != a.helpers.Window.MainHeight(a.ContextTree)-1 {
		t.Fatalf("at the end of the Review view, the thumb is on rows %v", got)
	}
}

func TestTheDiagramShowsTheSameBarForItsHeight(t *testing.T) {
	a := showBox(standInApp(40, 14), types.DiagramPlace{Level: "types", Component: "core", Package: "model", Box: "Order"})
	if got := mainThumb(a); len(got) == 0 || got[0] != 0 {
		t.Fatalf("at the top of the diagram, the thumb is on rows %v", got)
	}
	a = press(t, a, "G")
	if got := mainThumb(a); len(got) == 0 || got[len(got)-1] != a.helpers.Window.MainHeight(a.ContextTree)-1 {
		t.Fatalf("on the diagram's last box, the thumb is on rows %v", got)
	}
	a = showBox(standInApp(110, 40), types.DiagramPlace{Level: "components", Box: "core"})
	if got := mainThumb(a); got != nil {
		t.Fatalf("a diagram that fits shows a thumb on rows %v", got)
	}
}
