package gui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/models"
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
	state.Written = &models.Written{Title: "Fix the guide wording", Sections: []models.Section{{ID: "shape", Title: "Shape", Markdown: "Before.\n\n<svg viewBox=\"0 0 10 10\"><rect width=\"4\"/></svg>\n\nAfter."}}}
	a = a.with(a.helpers.Refresh.WithState(a.ContextTree, state))
	var text []string
	for _, l := range a.ReviewView.View.Lines {
		text = append(text, utils.Stripped(l.Text))
	}
	got := strings.Join(strings.Fields(strings.Join(text, " ")), " ")
	if strings.Contains(got, "<svg") || !strings.Contains(got, "A drawing is here that a terminal cannot show. gX opens the written review's drawings in the browser.") {
		t.Fatalf("the section reads:\n%s", got)
	}
}

func TestAWrittenReviewsDrawingsOpenInTheBrowser(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	a, visited := opened(t, "github.com/octo/widgets/30", 100, 30)
	a = press(t, a, "0", "g", "X")
	if len(*visited) != 0 || !strings.Contains(a.Notice, "this review has no drawings") {
		t.Fatalf("with no drawings gX opens %v and says %q", *visited, a.Notice)
	}
	state := a.Review.State
	state.Written = &models.Written{Title: "Fix <the> guide", Sections: []models.Section{
		{ID: "flow", Title: "The flow", Markdown: "Before.\n\n<svg viewBox=\"0 0 10 10\"><script>alert(1)</script><rect width=\"4\"/></svg>\n\nAfter."},
		{ID: "risks", Title: "Risks", Markdown: "None."},
	}}
	a = a.with(a.helpers.Refresh.WithState(a.ContextTree, state))
	a = press(t, a, "g", "X")
	if len(*visited) != 1 || !strings.HasSuffix((*visited)[0], "github.com_octo_widgets_30-drawings.html") {
		t.Fatalf("gX opened %v", *visited)
	}
	if a.Notice != "opened the written review's drawings in the browser" {
		t.Errorf("gX says %q", a.Notice)
	}
	page := readFile(t, (*visited)[0])
	for _, want := range []string{"<title>Drawings of Fix &lt;the&gt; guide</title>", "default-src 'none'", "<h2>The flow</h2>", `<svg viewBox="0 0 10 10">`} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q:\n%s", want, page)
		}
	}
	if strings.Contains(page, "Risks") {
		t.Errorf("a section with no drawing has a heading on the page:\n%s", page)
	}
}

func TestTheCursorStaysOnItsBlock(t *testing.T) {
	a := press(t, onGuide(t), "j", "j", "j")
	a = send(t, a, tea.WindowSizeMsg{Width: 90, Height: 30})
	if r := a.Diff.Rows[a.Diff.Cursor]; r.Block == nil || r.Block.StartLine != 3 || r.Block.Side != "RIGHT" {
		t.Fatalf("the cursor moved to %+v", r)
	}
}
