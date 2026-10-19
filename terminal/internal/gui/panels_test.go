package gui

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

func selectedKey(a App, n int) string {
	row, _ := a.Side(n).Selected()
	return row.Key
}

func TestTheAppOpensOnTheReviewPanelBesideTheReviewView(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	if a.Focus() != context.ReviewPanel || a.Screen() != context.ScreenReview {
		t.Fatalf("it opens with panel %d on screen %d", a.Focus(), a.Screen())
	}
	screenIs(t, a, "panels")
	a = press(t, a, "j", "j")
	if got := selectedKey(a, context.ReviewPanel); got != "pull:22" {
		t.Fatalf("j j in the Review panel chose %s", got)
	}
	a = press(t, a, "enter")
	if a.Focus() != context.MainPanel || a.Screen() != context.ScreenReview {
		t.Fatalf("Enter gives the Review view the keys, not panel %d", a.Focus())
	}
	a = press(t, a, "esc")
	if a.Focus() != context.ReviewPanel {
		t.Fatalf("Esc goes back to the Review panel, not %d", a.Focus())
	}
	a = press(t, a, "esc", "q")
	if a.Notice != "ZZ or :q quits" {
		t.Fatalf("q with nothing to close says how to quit, not %q", a.Notice)
	}
}

func TestTheFilesPanelShowsTheSelectedFilesDiff(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "2")
	if a.Focus() != context.FilesPanel || a.Screen() != context.ScreenDiff || !slices.Equal(a.FilesPanel.Chosen, []string{"component:root"}) {
		t.Fatalf("2 shows the first component's diff: panel %d, screen %d, %q", a.Focus(), a.Screen(), a.FilesPanel.Chosen)
	}
	a = press(t, a, "j", "j")
	if got := selectedKey(a, context.FilesPanel); got != "file:Sources/Basket/Basket.swift" {
		t.Fatalf("j j chose %s", got)
	}
	if got := cursorAt(a); got != "Sources/Basket/Basket.swift file" {
		t.Fatalf("the main panel follows the file, at %s", got)
	}
	screenIs(t, a, "files")
	a = press(t, a, "enter", "j")
	if a.Focus() != context.MainPanel || cursorAt(a) == "Sources/Basket/Basket.swift file" {
		t.Fatalf("Enter gives the diff the keys: panel %d at %s", a.Focus(), cursorAt(a))
	}
	a = press(t, a, "]", "f")
	if got := selectedKey(a, context.FilesPanel); got != "file:Sources/Basket/Basket.swift" || cursorAt(a) == "Sources/Catalog/MemoryCatalog.swift file" {
		t.Fatalf("]f in the diff of one file stays in it, the Files panel on %s, the diff at %s", got, cursorAt(a))
	}
	a = press(t, a, "esc", "g", "g", "z", "a")
	if a.Side(context.FilesPanel).Index("file:Sources/Basket/Basket.swift") >= 0 {
		t.Fatal("za on a component folds its files away")
	}
	a = press(t, a, "enter")
	if a.Side(context.FilesPanel).Index("file:Sources/Basket/Basket.swift") < 0 {
		t.Fatal("Enter on a folded component unfolds it")
	}
}

func TestTheNotesPanelShowsTheNoteInItsDiff(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "3")
	if got := selectedKey(a, context.NotesPanel); got != "queued:1" {
		t.Fatalf("3 chose %s", got)
	}
	if got := noteKeyAt(a); got != "notehead queued:1" {
		t.Fatalf("the main panel shows the note, on %s", got)
	}
	a = press(t, a, "j", "j")
	if got := noteKeyAt(a); got != "notehead conversation:1" {
		t.Fatalf("j j past the heading shows the conversation, on %s", got)
	}
	screenIs(t, a, "notes-panel")
	a = press(t, a, "k", "d", "d")
	if !a.Has(context.ConfirmationKey) || a.Confirmation.Comment != 2 {
		t.Fatalf("dd in the Notes panel asks to drop the comment chosen: %+v", a.Confirmation)
	}
	a = press(t, a, "n", "P")
	if !a.Has(context.PostingKey) || a.Posting.Pull != 22 {
		t.Fatalf("P posts for the chosen comment's pull request: %+v", a.Posting)
	}
}

func TestTheDiagramPanelListsTheBoxesInReadingOrder(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "4")
	if a.Screen() != context.ScreenDiagram || selectedKey(a, context.DiagramPanel) != "box:root" {
		t.Fatalf("4 shows the diagram: screen %d on %s", a.Screen(), selectedKey(a, context.DiagramPanel))
	}
	if hints := a.controllers.Hints(a.ContextTree); hints[1] != [2]string{"-", "Review view"} {
		t.Fatalf("on the components, the bar names - for the Review view, as the diagram's does: %v", hints)
	}
	a = press(t, a, "enter")
	if a.Focus() != context.MainPanel || a.Screen() != context.ScreenDiagram || a.Diagram.Level.Kind != "components" {
		t.Fatalf("Enter gives the diagram in the main panel the keys: panel %d, screen %d, %s", a.Focus(), a.Screen(), a.Diagram.Level.Kind)
	}
	a = press(t, a, "enter", "esc")
	if a.Focus() != context.DiagramPanel || a.Diagram.Level.Kind != "packages" {
		t.Fatalf("Enter there steps in, and Esc gives the panel back the keys on that level: %d, %s", a.Focus(), a.Diagram.Level.Kind)
	}
	before := a.Diagram.Focus
	a = press(t, a, "j")
	if a.Diagram.Focus == before || selectedKey(a, context.DiagramPanel) != presentation.BoxKey(a.Diagram.Level.Cards[a.Diagram.Focus].ID) {
		t.Fatalf("j moves the box and the panel together, to %s", selectedKey(a, context.DiagramPanel))
	}
	screenIs(t, a, "diagram-panel")
	a = press(t, a, "-")
	if a.Diagram.Level.Kind != "components" || a.Focus() != context.DiagramPanel {
		t.Fatalf("- steps out: %s, panel %d", a.Diagram.Level.Kind, a.Focus())
	}
	a = press(t, a, "0")
	if a.Focus() != context.MainPanel {
		t.Fatal("0 gives the main panel the keys")
	}
}

func TestHAndLWalkEveryPanelOnTheLeftAndGoRoundAtBothEnds(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	steps := []struct {
		key    string
		panel  int
		screen context.Screen
	}{
		{"l", context.FilesPanel, context.ScreenDiff}, {"l", context.NotesPanel, context.ScreenDiff}, {"l", context.DiagramPanel, context.ScreenDiagram},
		{"l", context.ReviewPanel, context.ScreenReview}, {"h", context.DiagramPanel, context.ScreenDiagram}, {"h", context.NotesPanel, context.ScreenDiff},
		{"h", context.FilesPanel, context.ScreenDiff}, {"h", context.ReviewPanel, context.ScreenReview},
	}
	for _, s := range steps {
		from := a.Focus()
		a = press(t, a, s.key)
		if a.Focus() != s.panel || a.Screen() != s.screen {
			t.Fatalf("%s from panel %d went to panel %d, screen %d", s.key, from, a.Focus(), a.Screen())
		}
	}
}

func TestInTheDiagramPanelJAndKMoveAndEnterGivesTheMainPanelsDiagramHJKL(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "4", "enter", "enter", "esc")
	focusedOn(t, a, "sources")
	for _, s := range []struct{ key, box string }{{"j", "web-src"}, {"k", "sources"}} {
		a = press(t, a, s.key)
		focusedOn(t, a, s.box)
		if a.Focus() != context.DiagramPanel || selectedKey(a, context.DiagramPanel) != presentation.BoxKey(s.box) {
			t.Fatalf("%s keeps the Diagram panel and its row on the box: panel %d, %s", s.key, a.Focus(), selectedKey(a, context.DiagramPanel))
		}
	}
	if hints := a.controllers.Hints(a.ContextTree); hints[0] != [2]string{"Enter", "Diagram"} || hints[2] != [2]string{"h l", "Panels"} {
		t.Fatalf("the bar names Enter for the diagram and h l for the panels: %v", hints)
	}
	listed := press(t, a, "space", "?").Menu.Rows
	if listed[2].Left != "Ctrl-d Ctrl-u" || listed[2].Right != "half a page down or up" || listed[3].Left != "Ctrl-f Ctrl-b" {
		t.Fatalf("Space ? lists the pages as %q: %q, then %q", listed[2].Left, listed[2].Right, listed[3].Left)
	}
	if listed[4].Left != "h l" || listed[4].Right != "the panel above or below; l goes round to Review" {
		t.Fatalf("Space ? lists h l as %q: %q", listed[4].Left, listed[4].Right)
	}
	if listed[5].Left != "Enter" || listed[5].Right != "the diagram in the main panel, which moves through its rows of boxes and steps in" {
		t.Fatalf("Space ? lists Enter as %q: %q", listed[5].Left, listed[5].Right)
	}
	a = press(t, a, "enter")
	if a.Focus() != context.MainPanel || a.Screen() != context.ScreenDiagram {
		t.Fatalf("Enter gives the main panel's diagram the keys: panel %d, screen %d", a.Focus(), a.Screen())
	}
	for _, s := range []struct{ key, box string }{{"l", "web-src"}, {"l", "web-src"}, {"h", "sources"}, {"h", "sources"}} {
		a = press(t, a, s.key)
		focusedOn(t, a, s.box)
		if a.Focus() != context.MainPanel || selectedKey(a, context.DiagramPanel) != presentation.BoxKey(s.box) {
			t.Fatalf("%s in the main panel moves to the box, and the Diagram panel's row follows: panel %d, %s", s.key, a.Focus(), selectedKey(a, context.DiagramPanel))
		}
	}
}

func TestBiggerAndSmallerScreenModes(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "2", "j", "j", "enter")
	normal := a.helpers.Window.MainWidth(a.ContextTree)
	a = press(t, a, "+")
	if half := a.helpers.Window.MainWidth(a.ContextTree); half <= normal {
		t.Fatalf("+ makes the main panel wider: %d, then %d", normal, half)
	}
	a = press(t, a, "+")
	if a.helpers.Window.MainWidth(a.ContextTree) != 98 || a.helpers.Window.Layout(a.ContextTree).Sides[context.FilesPanel].Shown() {
		t.Fatalf("+ again gives the main panel the whole screen: %d", a.helpers.Window.MainWidth(a.ContextTree))
	}
	screenIs(t, a, "full")
	a = press(t, a, "_", "_")
	if a.helpers.Window.MainWidth(a.ContextTree) != normal {
		t.Fatalf("_ _ goes back: %d", a.helpers.Window.MainWidth(a.ContextTree))
	}
}

func TestANarrowTerminalShowsOnlyThePanelWithTheKeys(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 50, 12)
	l := a.helpers.Window.Layout(a.ContextTree)
	if !l.Sides[context.ReviewPanel].Shown() || l.Main.Shown() || l.Sides[context.FilesPanel].Shown() {
		t.Fatalf("at 50 by 12 only the Review panel shows: %+v", l)
	}
	a = press(t, a, "2", "j", "j", "enter")
	if l := a.helpers.Window.Layout(a.ContextTree); !l.Main.Shown() || l.Main.W != 50 {
		t.Fatalf("with the keys, the main panel takes the screen: %+v", l.Main)
	}
}

func TestAClickChoosesARowAndADoubleClickOpensIt(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	files := a.helpers.Window.Layout(a.ContextTree).Sides[context.FilesPanel].Inner()
	click := tea.MouseMsg{X: files.X + 4, Y: files.Y + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	a = send(t, a, click)
	if a.Focus() != context.FilesPanel || selectedKey(a, context.FilesPanel) != "file:Sources/Basket/Basket.swift" {
		t.Fatalf("a click chose panel %d, %s", a.Focus(), selectedKey(a, context.FilesPanel))
	}
	a = send(t, a, click)
	if a.Focus() != context.MainPanel || a.Screen() != context.ScreenDiff {
		t.Fatalf("a double click opens the file's diff: panel %d", a.Focus())
	}
}

func TestTheKeysOfAPanelAndTheCommandLine(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "space")
	if !strings.Contains(a.View(), "Keys: Space ?") {
		t.Fatalf("after the leader, the bar names the keys that can follow:\n%s", a.View())
	}
	a = press(t, a, "?")
	if !a.MenuOpen() || a.Menu.Rows[0].Heading != "Review" || a.Menu.Rows[a.Menu.Cursor].Left != "j k gg G" {
		t.Fatalf("Space ? lists the Review panel's keys first: %+v", a.Menu)
	}
	screenIs(t, a, "keys")
	a = press(t, a, "q")
	if a.MenuOpen() {
		t.Fatal("q did not close the keys")
	}
	a = press(t, a, ":")
	a = typed(t, a, "frob")
	if a.Notice != "not a command: :frob" || !a.Bad {
		t.Fatalf("an unknown command says so, not %q", a.Notice)
	}
	a = press(t, a, ":")
	for _, r := range "q" {
		a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if _, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil || cmd() != tea.Quit() {
		t.Fatal(":q did not quit")
	}
}

// In Space ? and every list that pops up, j, k, gg, G, half pages and
// pages move the list, after a count too, and leave the diff under it as
// it was. A key the list does not take closes it and goes on to the diff,
// with the count typed before it, as 2]c.
func TestAListThatPopsUpTakesItsOwnMoves(t *testing.T) {
	under := press(t, onBasket(t), "j", "j")
	at := cursorAt(under)
	a := press(t, under, "space", "?")
	rows, pn := a.helpers.Window.MenuRows(a.ContextTree), a.Menu
	steps := []struct {
		keys []string
		want int
	}{
		{[]string{"G"}, pn.Nth(len(pn.Rows), rows).Cursor},
		{[]string{"g", "g"}, pn.Nth(1, rows).Cursor},
		{[]string{"5", "j"}, pn.Nth(6, rows).Cursor},
		{[]string{"ctrl+d"}, pn.Nth(6, rows).Move(rows/2, rows).Cursor},
		{[]string{"ctrl+u", "k"}, pn.Nth(5, rows).Cursor},
		{[]string{"1", "2", "G"}, pn.Nth(12, rows).Cursor},
	}
	for _, s := range steps {
		a = press(t, a, s.keys...)
		if !a.MenuOpen() || a.Menu.Cursor != s.want || cursorAt(a) != at {
			t.Fatalf("%v in Space ? left the list open %v on row %d, not %d, and the diff at %s", s.keys, a.MenuOpen(), a.Menu.Cursor, s.want, cursorAt(a))
		}
	}
	a = press(t, a, "2", "]", "c")
	if want := cursorAt(press(t, under, "2", "]", "c")); a.MenuOpen() || cursorAt(a) != want {
		t.Fatalf("2]c in Space ? left the list open %v and the diff at %s, not %s", a.MenuOpen(), cursorAt(a), want)
	}
}

func TestAReviewThatWillNotOpenSaysWhyAndTriesAgain(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/99", 80, 20)
	if a.Review != nil || !strings.HasPrefix(a.Failed, "could not open github.com/octo/widgets/99") {
		t.Fatalf("a review the server does not have: %q", a.Failed)
	}
	screenIs(t, a, "will-not-open")
	a = press(t, a, "R")
	if a.Review != nil || a.Failed == "" || a.Loading {
		t.Fatalf("R tries again, and says why again: %q", a.Failed)
	}
}

func TestAJumpGivesTheMainPanelTheKeys(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "space", "m", "2")
	if a.Focus() != context.MainPanel || a.Keys.Typed() != "2" {
		t.Fatalf("in the main panel, 2 starts a count: panel %d, typed %q", a.Focus(), a.Keys.Typed())
	}
	a = press(t, a, "esc", "2")
	if a.Focus() != context.FilesPanel || a.Screen() != context.ScreenDiff {
		t.Fatalf("Esc, then 2, shows the Files panel's diff: panel %d, screen %d", a.Focus(), a.Screen())
	}
	a = press(t, a, "ctrl+o")
	if a.Focus() != context.MainPanel || a.Screen() != context.ScreenReview || a.Driving() != context.ReviewPanel {
		t.Fatalf("Ctrl-o goes back to the Review view with its keys: panel %d, screen %d", a.Focus(), a.Screen())
	}
}

func TestOnAShortScreenThePanelsFoldToStripsAndEveryRowKeepsItsWidth(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 12)
	lines := strings.Split(a.View(), "\n")
	for i, l := range lines {
		if w := ansi.StringWidth(l); w != 100 {
			t.Fatalf("row %d is %d wide: %q", i, w, l)
		}
	}
	for _, title := range []string{"──[3]─Notes", "──[4]─Diagram"} {
		if !strings.Contains(a.View(), title) {
			t.Fatalf("no strip %q on a short screen", title)
		}
	}
}
