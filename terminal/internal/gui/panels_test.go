package gui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
)

func selectedKey(a App, n int) string {
	row, _ := a.sides[n].selected()
	return row.Key
}

func TestTheAppOpensOnTheReviewPanelBesideTheReviewView(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	if a.focus != reviewPanel || a.screen != screenReview {
		t.Fatalf("it opens with panel %d on screen %d", a.focus, a.screen)
	}
	screenIs(t, a, "panels")
	a = press(t, a, "j")
	if got := selectedKey(a, reviewPanel); got != "pull:22" {
		t.Fatalf("j in the Review panel chose %s", got)
	}
	a = press(t, a, "enter")
	if a.focus != mainPanel || a.screen != screenReview {
		t.Fatalf("Enter gives the Review view the keys, not panel %d", a.focus)
	}
	a = press(t, a, "esc")
	if a.focus != reviewPanel {
		t.Fatalf("Esc goes back to the Review panel, not %d", a.focus)
	}
	a = press(t, a, "q")
	if a.notice != "ZZ or :q quits" {
		t.Fatalf("q with nothing to close says how to quit, not %q", a.notice)
	}
}

func TestTheFilesPanelShowsTheSelectedFilesDiff(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "2")
	if a.focus != filesPanel || a.screen != screenDiff || a.diff.Component != "root" {
		t.Fatalf("2 shows the first component's diff: panel %d, screen %d, %q", a.focus, a.screen, a.diff.Component)
	}
	a = press(t, a, "j", "j")
	if got := selectedKey(a, filesPanel); got != "file:Sources/Basket/Basket.swift" {
		t.Fatalf("j j chose %s", got)
	}
	if got := cursorAt(a); got != "Sources/Basket/Basket.swift file" {
		t.Fatalf("the main panel follows the file, at %s", got)
	}
	screenIs(t, a, "files")
	a = press(t, a, "enter", "j")
	if a.focus != mainPanel || cursorAt(a) == "Sources/Basket/Basket.swift file" {
		t.Fatalf("Enter gives the diff the keys: panel %d at %s", a.focus, cursorAt(a))
	}
	a = press(t, a, "]", "f")
	if got := selectedKey(a, filesPanel); got != "file:Sources/Catalog/MemoryCatalog.swift" {
		t.Fatalf("]f in the diff moves the Files panel with it, to %s", got)
	}
	a = press(t, a, "esc", "g", "g", "z", "a")
	if a.sides[filesPanel].index("file:Sources/Basket/Basket.swift") >= 0 {
		t.Fatal("za on a component folds its files away")
	}
	a = press(t, a, "enter")
	if a.sides[filesPanel].index("file:Sources/Basket/Basket.swift") < 0 {
		t.Fatal("Enter on a folded component unfolds it")
	}
}

func TestTheNotesPanelShowsTheNoteInItsDiff(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "3")
	if got := selectedKey(a, notesPanel); got != "queued:1" {
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
	if a.confirming == nil || a.confirming.comment != 2 {
		t.Fatalf("dd in the Notes panel asks to drop the comment chosen: %+v", a.confirming)
	}
	a = press(t, a, "n", "P")
	if a.posting == nil || a.posting.pull != 22 {
		t.Fatalf("P posts for the chosen comment's pull request: %+v", a.posting)
	}
}

func TestTheDiagramPanelListsTheBoxesInReadingOrder(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "4")
	if a.screen != screenDiagram || selectedKey(a, diagramPanel) != "box:root" {
		t.Fatalf("4 shows the diagram: screen %d on %s", a.screen, selectedKey(a, diagramPanel))
	}
	a = press(t, a, "enter")
	if a.focus != diagramPanel || a.dg.Level.Kind != "packages" {
		t.Fatalf("Enter steps in and keeps the panel: %d, %s", a.focus, a.dg.Level.Kind)
	}
	before := a.dg.Focus
	a = press(t, a, "j")
	if a.dg.Focus == before || selectedKey(a, diagramPanel) != presentation.BoxKey(a.dg.Level.Cards[a.dg.Focus].ID) {
		t.Fatalf("j moves the box and the panel together, to %s", selectedKey(a, diagramPanel))
	}
	screenIs(t, a, "diagram-panel")
	a = press(t, a, "-")
	if a.dg.Level.Kind != "components" || a.focus != diagramPanel {
		t.Fatalf("- steps out: %s, panel %d", a.dg.Level.Kind, a.focus)
	}
	a = press(t, a, "0")
	if a.focus != mainPanel {
		t.Fatal("0 gives the main panel the keys")
	}
}

func TestHAndLGoToThePanelAboveOrBelowAndHGoesRoundFromTheFirst(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	steps := []struct {
		key    string
		panel  int
		screen screen
	}{
		{"l", filesPanel, screenDiff}, {"l", notesPanel, screenDiff}, {"h", filesPanel, screenDiff},
		{"h", reviewPanel, screenReview}, {"h", diagramPanel, screenDiagram},
	}
	for _, s := range steps {
		from := a.focus
		a = press(t, a, s.key)
		if a.focus != s.panel || a.screen != s.screen {
			t.Fatalf("%s from panel %d went to panel %d, screen %d", s.key, from, a.focus, a.screen)
		}
	}
	a = press(t, a, "3", "l")
	if a.focus != diagramPanel || a.screen != screenDiagram {
		t.Fatalf("l from Notes enters Diagram, not panel %d", a.focus)
	}
}

func TestHAndLInTheDiagramPanelGoToTheBoxLeftOrRight(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "4", "enter")
	focusedOn(t, a, "sources")
	for _, s := range []struct{ key, box string }{{"l", "web-src"}, {"l", "web-src"}, {"h", "sources"}, {"h", "sources"}} {
		a = press(t, a, s.key)
		focusedOn(t, a, s.box)
		if a.focus != diagramPanel || selectedKey(a, diagramPanel) != presentation.BoxKey(s.box) {
			t.Fatalf("%s keeps the Diagram panel and its row on the box: panel %d, %s", s.key, a.focus, selectedKey(a, diagramPanel))
		}
	}
	if a.hints()[0] != [2]string{"h j k l", "Move"} {
		t.Fatalf("the bar names the keys that move: %v", a.hints()[0])
	}
	if listed := press(t, a, "space", "?").menu.Rows[2]; listed.Left != "h l" || listed.Right != "the nearest box left or right" {
		t.Fatalf("Space ? lists h l as %q: %q", listed.Left, listed.Right)
	}
	a = press(t, a, "3")
	if a.focus != notesPanel {
		t.Fatalf("3 leaves the Diagram panel for Notes, not panel %d", a.focus)
	}
}

func TestBiggerAndSmallerScreenModes(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "2", "j", "j", "enter")
	normal := a.mainWidth()
	a = press(t, a, "+")
	if half := a.mainWidth(); half <= normal {
		t.Fatalf("+ makes the main panel wider: %d, then %d", normal, half)
	}
	a = press(t, a, "+")
	if a.mainWidth() != 98 || a.layout().sides[filesPanel].Shown() {
		t.Fatalf("+ again gives the main panel the whole screen: %d", a.mainWidth())
	}
	screenIs(t, a, "full")
	a = press(t, a, "_", "_")
	if a.mainWidth() != normal {
		t.Fatalf("_ _ goes back: %d", a.mainWidth())
	}
}

func TestANarrowTerminalShowsOnlyThePanelWithTheKeys(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 50, 12)
	l := a.layout()
	if !l.sides[reviewPanel].Shown() || l.main.Shown() || l.sides[filesPanel].Shown() {
		t.Fatalf("at 50 by 12 only the Review panel shows: %+v", l)
	}
	a = press(t, a, "2", "j", "j", "enter")
	if l := a.layout(); !l.main.Shown() || l.main.W != 50 {
		t.Fatalf("with the keys, the main panel takes the screen: %+v", l.main)
	}
}

func TestAClickChoosesARowAndADoubleClickOpensIt(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	files := a.layout().sides[filesPanel].Inner()
	click := tea.MouseMsg{X: files.X + 4, Y: files.Y + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	a = send(t, a, click)
	if a.focus != filesPanel || selectedKey(a, filesPanel) != "file:Sources/Basket/Basket.swift" {
		t.Fatalf("a click chose panel %d, %s", a.focus, selectedKey(a, filesPanel))
	}
	a = send(t, a, click)
	if a.focus != mainPanel || a.screen != screenDiff {
		t.Fatalf("a double click opens the file's diff: panel %d", a.focus)
	}
}

func TestTheKeysOfAPanelAndTheCommandLine(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "space")
	if !strings.Contains(a.View(), "Keys: Space ?") {
		t.Fatalf("after the leader, the bar names the keys that can follow:\n%s", a.View())
	}
	a = press(t, a, "?")
	if a.menu == nil || a.menu.Rows[0].Heading != "Review" || a.menu.Rows[a.menu.Cursor].Left != "j k gg G" {
		t.Fatalf("Space ? lists the Review panel's keys first: %+v", a.menu)
	}
	screenIs(t, a, "keys")
	a = press(t, a, "q")
	if a.menu != nil {
		t.Fatal("q did not close the keys")
	}
	a = press(t, a, ":")
	a = typed(t, a, "frob")
	if a.notice != "not a command: :frob" || !a.bad {
		t.Fatalf("an unknown command says so, not %q", a.notice)
	}
	a = press(t, a, ":")
	for _, r := range "q" {
		a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if _, cmd := a.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil || cmd() != tea.Quit() {
		t.Fatal(":q did not quit")
	}
}

func TestAReviewThatWillNotOpenSaysWhyAndTriesAgain(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/99", 80, 20)
	if a.review != nil || !strings.HasPrefix(a.failed, "could not open github.com/octo/widgets/99") {
		t.Fatalf("a review the server does not have: %q", a.failed)
	}
	screenIs(t, a, "will-not-open")
	a = press(t, a, "R")
	if a.review != nil || a.failed == "" || a.loading {
		t.Fatalf("R tries again, and says why again: %q", a.failed)
	}
}

func TestAJumpGivesTheMainPanelTheKeys(t *testing.T) {
	a, _ := opened(t, "github.com/octo/widgets/21", 100, 30)
	a = press(t, a, "space", "m", "2")
	if a.focus != mainPanel || a.keys.Typed() != "2" {
		t.Fatalf("in the main panel, 2 starts a count: panel %d, typed %q", a.focus, a.keys.Typed())
	}
	a = press(t, a, "esc", "2")
	if a.focus != filesPanel || a.screen != screenDiff {
		t.Fatalf("Esc, then 2, shows the Files panel's diff: panel %d, screen %d", a.focus, a.screen)
	}
	a = press(t, a, "ctrl+o")
	if a.focus != mainPanel || a.screen != screenReview || a.driving != reviewPanel {
		t.Fatalf("Ctrl-o goes back to the Review view with its keys: panel %d, screen %d", a.focus, a.screen)
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
