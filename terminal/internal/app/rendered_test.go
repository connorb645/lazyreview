package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"review-desk/terminal/internal/server"
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
	if a.diff.rendered["docs/guide.md"] {
		t.Fatal("Space r goes back to the diff")
	}
}

func TestAFileWithNoRenderedViewSaysSo(t *testing.T) {
	a := press(t, onBasket(t), "space", "r")
	if !a.bad || a.notice != "Space r shows a Markdown file rendered, and this file has no rendered view" {
		t.Fatalf("the status line says %q", a.notice)
	}
}

func TestASelectionOfBlocksIsCommentedOnByItsSourceLines(t *testing.T) {
	editor, seen := writer(t, "Why every item?")
	a := onGuide(t, editor)
	a = press(t, a, "j", "j", "V", "j")
	screenIs(t, a, "rendered-visual")
	a = press(t, a, "c")
	if a.bad {
		t.Fatalf("the status line says %q", a.notice)
	}
	file := readFile(t, seen)
	if !strings.Contains(file, "# Goes on #30, removed line 3 to line 3.\n#\n#      3 - The catalog lists items.\n#      3 + The catalog lists every item.\n") {
		t.Fatalf("the editor was given:\n%s", file)
	}
	if n, ok := noteOf(*a.review, "docs/guide.md", "queued:2"); !ok || n.at.StartSide != "LEFT" {
		t.Fatalf("the comment on the passage: %+v", n.at)
	}
	if r := a.diff.rows[a.diff.cursor]; r.block == nil || r.block.StartLine != 3 || r.block.Side != "RIGHT" {
		t.Fatalf("after queueing, the cursor is on %+v", r)
	}
}

func TestAWrittenReviewsDiagramIsSaidNotDrawn(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	state := a.review.state
	state.Written = &server.Written{Title: "Fix the guide wording", Sections: []server.Section{{ID: "shape", Title: "Shape", Markdown: "Before.\n\n<svg viewBox=\"0 0 10 10\"><rect width=\"4\"/></svg>\n\nAfter."}}}
	a = a.withState(state)
	var text []string
	for _, l := range a.rv.view.lines {
		text = append(text, stripped(l.text))
	}
	got := strings.Join(strings.Fields(strings.Join(text, " ")), " ")
	if strings.Contains(got, "<svg") || !strings.Contains(got, "A diagram is here that a terminal cannot draw. gX opens this review in the web app.") {
		t.Fatalf("the section reads:\n%s", got)
	}
}

func TestTheCursorStaysOnItsBlock(t *testing.T) {
	a := press(t, onGuide(t), "j", "j", "j")
	a = send(t, a, tea.WindowSizeMsg{Width: 90, Height: 30})
	if r := a.diff.rows[a.diff.cursor]; r.block == nil || r.block.StartLine != 3 || r.block.Side != "RIGHT" {
		t.Fatalf("the cursor moved to %+v", r)
	}
}
