package gui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func onGuide(t *testing.T, set ...func(*Options)) App {
	t.Helper()
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30, set...)
	a = typed(t, press(t, a, "space", "f"), "docs/guide")
	if cursorAt(a) != "docs/guide.md file" {
		t.Fatalf("the finder opened %s", cursorAt(a))
	}
	return press(t, a, "space", "r")
}

func TestAMarkdownFileShowsRendered(t *testing.T) {
	a := onGuide(t)
	screenIs(t, a, "rendered")
	a = press(t, a, "space", "r")
	if a.Diff.Rendered["docs/guide.md"] {
		t.Fatal("Space r goes back to the diff")
	}
}

func TestAFileWithNoRenderedViewSaysSo(t *testing.T) {
	a := press(t, onBasket(t), "space", "r")
	if !a.Bad || a.Notice != "Space r shows a Markdown file rendered, and this file has no rendered view" {
		t.Fatalf("the status line says %q", a.Notice)
	}
}

func TestASelectionOfBlocksIsCommentedOnByItsSourceLines(t *testing.T) {
	editor, seen := writer(t, "Why every item?")
	a := onGuide(t, editor)
	a = press(t, a, "j", "j", "V", "j")
	screenIs(t, a, "rendered-visual")
	a = press(t, a, "c")
	if a.Bad {
		t.Fatalf("the status line says %q", a.Notice)
	}
	file := readFile(t, seen)
	if !strings.Contains(file, "# Goes on #30, removed line 3 to line 3.\n#\n#      3 - The catalog lists items.\n#      3 + The catalog lists every item.\n") {
		t.Fatalf("the editor was given:\n%s", file)
	}
	if n, ok := presentation.NoteOf(*a.Review, "docs/guide.md", "queued:2"); !ok || n.At.StartSide != "LEFT" {
		t.Fatalf("the comment on the passage: %+v", n.At)
	}
	if r := a.Diff.Rows[a.Diff.Cursor]; r.Block == nil || r.Block.StartLine != 3 || r.Block.Side != "RIGHT" {
		t.Fatalf("after queueing, the cursor is on %+v", r)
	}
}

func TestAWrittenReviewsDiagramIsSaidNotDrawn(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	state := a.Review.State
	state.Written = &server.Written{Title: "Fix the guide wording", Sections: []server.Section{{ID: "shape", Title: "Shape", Markdown: "Before.\n\n<svg viewBox=\"0 0 10 10\"><rect width=\"4\"/></svg>\n\nAfter."}}}
	a = a.with(a.helpers.Refresh.WithState(a.ContextTree, state))
	var text []string
	for _, l := range a.ReviewView.View.Lines {
		text = append(text, utils.Stripped(l.Text))
	}
	got := strings.Join(strings.Fields(strings.Join(text, " ")), " ")
	if strings.Contains(got, "<svg") || !strings.Contains(got, "A diagram is here that a terminal cannot draw. gX opens this review in the web app.") {
		t.Fatalf("the section reads:\n%s", got)
	}
}

func TestTheCursorStaysOnItsBlock(t *testing.T) {
	a := press(t, onGuide(t), "j", "j", "j")
	a = send(t, a, tea.WindowSizeMsg{Width: 90, Height: 30})
	if r := a.Diff.Rows[a.Diff.Cursor]; r.Block == nil || r.Block.StartLine != 3 || r.Block.Side != "RIGHT" {
		t.Fatalf("the cursor moved to %+v", r)
	}
}
