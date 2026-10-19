package presentation

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/config"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
)

func TestALineCutAtAnOffsetMarksWhatIsHiddenOnEitherSide(t *testing.T) {
	for _, c := range []struct {
		text        string
		left, width int
		want        string
	}{
		{"abcdefghij", 3, 5, "…efg…"},
		{"abcdefghij", 5, 5, "…ghij"},
		{"abcdefghij", 6, 5, "…hij"},
		{"ab", 5, 5, "…"},
		{"", 5, 5, ""},
		{"日本語のテキスト", 2, 7, "… 語の…"},
	} {
		if got := shifted(c.text, c.left, c.width); got != c.want || ansi.StringWidth(got) > c.width {
			t.Errorf("%q from column %d in %d is %q, not %q", c.text, c.left, c.width, got, c.want)
		}
	}
}

// longLines is a unified diff 40 columns wide, its code 26 columns, of
// three lines: 60, 10 and 46 columns long.
func longLines() DiffView {
	line := func(n int, text string) DiffRow {
		return DiffRow{Kind: "line", Path: "a.ts", Line: &server.Row{Kind: "same", Old: n, New: n, Text: text}}
	}
	rows := []DiffRow{
		{Kind: "file", Path: "a.ts"},
		line(1, strings.Repeat("a", 60)),
		line(2, strings.Repeat("b", 10)),
		line(3, strings.Repeat("c", 46)),
	}
	return DiffView{Rows: rows, Width: 40}
}

func TestTheCodeScrollsUntilTheWidestLineShownEndsAtTheRightEdge(t *testing.T) {
	v := longLines()
	if got := ScrollEnd(v, 10); got != 34 {
		t.Fatalf("with every line shown, the code scrolls 34 columns, not %d", got)
	}
	v.Top = 2
	if got := ScrollEnd(v, 10); got != 20 {
		t.Fatalf("with the longest line scrolled off, it scrolls 20, not %d", got)
	}
	if got := ScrollEnd(v, 1); got != 0 {
		t.Fatalf("with only a short line shown, it does not scroll: %d", got)
	}
	v.Top, v.Split = 0, true
	if got := ScrollEnd(v, 10); got != 49 {
		t.Fatalf("a split side has 11 columns of code, so it scrolls 49, not %d", got)
	}
}

func TestAScrolledLineKeepsItsGutterAndHeaderAndMarksItsLeft(t *testing.T) {
	v := longLines()
	v.Width, v.Left = 40, 34
	th := theme.New(config.Colours{}, true, termenv.Ascii)
	lines := DiffLines(i18n.EnglishTranslationSet(), th, KeyLabels{}, types.ReviewData{}, v, 10, nil, true)
	if got := ansi.Strip(lines[1]); got != "     1    1   …"+strings.Repeat("a", 25) {
		t.Fatalf("the longest line ends at the right edge: %q", got)
	}
	if got := ansi.Strip(lines[2]); got != "     2    2   …" {
		t.Fatalf("a line scrolled out of sight leaves its mark: %q", got)
	}
	if got := ansi.Strip(lines[0]); !strings.Contains(got, "a.ts") {
		t.Fatalf("the file's header stays put: %q", got)
	}
}

func TestBothSidesOfASplitRowScrollTogether(t *testing.T) {
	old, now := strings.Repeat("o", 40), strings.Repeat("n", 40)
	v := DiffView{Width: 40, Split: true, Left: 10, Rows: []DiffRow{{Kind: "line", Path: "a.ts",
		Left: &server.Row{Kind: "del", Old: 1, Text: old}, Right: &server.Row{Kind: "add", New: 1, Text: now}}}}
	th := theme.New(config.Colours{}, true, termenv.Ascii)
	line := ansi.Strip(DiffLines(i18n.EnglishTranslationSet(), th, KeyLabels{}, types.ReviewData{}, v, 10, nil, true)[0])
	if want := "   1 - …" + strings.Repeat("o", 9) + "…" + "│   1 + …" + strings.Repeat("n", 9) + "…"; !strings.Contains(line, want) {
		t.Fatalf("both sides start at column 10, each marked on both ends: %q", line)
	}
}
