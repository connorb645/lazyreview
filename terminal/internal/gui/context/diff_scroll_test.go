package context

import (
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/models"
)

// wideDiff is a unified diff 40 columns wide, 26 of them code, of lines 60,
// 10 and 46 columns long.
func wideDiff() DiffContext {
	line := func(n int, text string) presentation.DiffRow {
		return presentation.DiffRow{Kind: "line", Path: "a.ts", Line: &models.Row{Kind: "same", Old: n, New: n, Text: text}}
	}
	return DiffContext{DiffView: presentation.DiffView{Width: 40, Rows: []presentation.DiffRow{
		{Kind: "file", Path: "a.ts"}, line(1, strings.Repeat("a", 60)), line(2, strings.Repeat("b", 10)), line(3, strings.Repeat("c", 46)),
	}}}
}

func TestSidewaysStopsAtTheFirstColumnAndWhereTheWidestLineEnds(t *testing.T) {
	v := wideDiff()
	for _, step := range []struct{ by, want int }{{30, 30}, {30, 34}, {5, 34}, {-10, 24}, {-100, 0}, {-5, 0}} {
		if v = v.Sideways(step.by, 10); v.Left != step.want {
			t.Fatalf("by %d the code is scrolled %d columns, not %d", step.by, v.Left, step.want)
		}
	}
}

func TestAMoveKeepsTheScrollEvenPastWhatNowShows(t *testing.T) {
	v := wideDiff().Sideways(34, 10)
	v.Top = 2
	if v = v.Sideways(5, 10); v.Left != 34 {
		t.Fatalf("with shorter lines shown, l leaves the scroll where it was, not at %d", v.Left)
	}
	if v = v.Sideways(-10, 10); v.Left != 24 {
		t.Fatalf("h scrolls back from there, to %d", v.Left)
	}
}
