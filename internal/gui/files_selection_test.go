package gui

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/gui/context"
)

const (
	basket  = "Sources/Basket/Basket.swift"
	memory  = "Sources/Catalog/MemoryCatalog.swift"
	search  = "Sources/Catalog/Search.swift"
	cart    = "web/src/cart.ts"
	catalog = "Sources/Catalog/Catalog.swift"
)

// shown are the files the main panel's diff shows, in its order.
func shown(a App) []string {
	var out []string
	for _, r := range a.Diff.Rows {
		if r.Kind == "file" {
			out = append(out, r.Path)
		}
	}
	return out
}

func filesFoot(a App) string {
	lines := strings.Split(ansi.Strip(a.View()), "\n")
	r := a.helpers.Window.Layout(a.ContextTree).Sides[context.FilesPanel]
	return lines[r.Y+r.H-1]
}

func TestTheDiffShowsOnlyTheFilesOfTheFilesPanelsRow(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	for _, step := range []struct {
		keys []string
		row  string
		want []string
	}{
		{[]string{"2"}, "component:root", []string{basket, memory, search, cart}},
		{[]string{"j"}, "dir:root:Sources/Basket", []string{basket}},
		{[]string{"j"}, "file:" + basket, []string{basket}},
		{[]string{"j"}, "dir:root:Sources/Catalog", []string{memory, search}},
		{[]string{"j"}, "file:" + memory, []string{memory}},
		{[]string{"G"}, "file:" + catalog, []string{catalog}},
		{[]string{"k"}, "component:context", []string{catalog}},
	} {
		a = press(t, a, step.keys...)
		if got := selectedKey(a, context.FilesPanel); got != step.row || !slices.Equal(shown(a), step.want) {
			t.Fatalf("%v: on %s the diff shows %v, not %v", step.keys, got, shown(a), step.want)
		}
	}
}

func TestFileJumpsStayAmongTheFilesShown(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "2", "j", "j", "j", "0", "[", "f")
	if got := cursorAt(a); got != memory+" file" {
		t.Fatalf("[f at the first file shown stays on it, not %s", got)
	}
	a = press(t, a, "]", "f", "]", "f")
	if got := cursorAt(a); got != search+" file" {
		t.Fatalf("]f at the last file shown stays on it, not %s", got)
	}
	if got := selectedKey(a, context.FilesPanel); got != "dir:root:Sources/Catalog" || !slices.Equal(a.FilesPanel.Chosen, []string{"dir:root:Sources/Catalog"}) {
		t.Fatalf("the Files panel stays on the folder that holds the diff's file, not %s: %v", got, a.FilesPanel.Chosen)
	}
	if got := ansi.Strip(a.mainTitle(200)); got != "[0]─Diff · widgets › "+search {
		t.Fatalf("the main panel's title is %q", got)
	}
	a = press(t, a, "esc")
	if !slices.Equal(shown(a), []string{memory, search}) {
		t.Fatalf("back in the Files panel, the diff keeps the folder until the cursor moves: %v", shown(a))
	}
	if a = press(t, a, "j"); !slices.Equal(shown(a), []string{memory}) {
		t.Fatalf("a move chooses the next row, MemoryCatalog, not %v", shown(a))
	}
}

func TestVSelectsFilesAcrossFoldersAndComponents(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "2", "j", "j", "v")
	if foot := filesFoot(a); !strings.Contains(foot, "VISUAL · 3 of 10") {
		t.Fatalf("while the selection grows the Files panel's foot says so: %q", foot)
	}
	if got := barSays(a); !strings.HasPrefix(got, "Grow, shrink: j k | Stop: Esc | Panels: h l") {
		t.Fatalf("the bar offers growing and stopping: %q", got)
	}
	for _, step := range []struct {
		keys []string
		want []string
	}{
		{[]string{"j"}, []string{basket, memory, search}},
		{[]string{"j"}, []string{basket, memory, search}},
		{[]string{"k", "k"}, []string{basket}},
		{[]string{"k", "k"}, []string{basket, memory, search, cart}},
	} {
		if a = press(t, a, step.keys...); !slices.Equal(shown(a), step.want) {
			t.Fatalf("%v: the selection shows %v, not %v", step.keys, shown(a), step.want)
		}
	}
	if a = press(t, a, "v"); a.FilesPanel.GrowFrom != "" || !slices.Equal(a.FilesPanel.Chosen, []string{"component:root"}) || strings.Contains(filesFoot(a), "VISUAL") {
		t.Fatalf("v stops the selection on the cursor's row: %q %v", a.FilesPanel.GrowFrom, a.FilesPanel.Chosen)
	}
	a = press(t, a, "G", "k", "k", "v", "j", "j")
	if !slices.Equal(shown(a), []string{cart, catalog}) {
		t.Fatalf("a selection across two components shows %v", shown(a))
	}
	a = press(t, a, "0")
	if a.FilesPanel.GrowFrom != "" || !slices.Equal(shown(a), []string{cart, catalog}) {
		t.Fatalf("with the keys gone the selection holds, no longer growing: %q %v", a.FilesPanel.GrowFrom, shown(a))
	}
	if got := ansi.Strip(a.mainTitle(200)); got != "[0]─Diff · Unchanged code › "+catalog {
		t.Fatalf("the title names the component of the file at the cursor: %q", got)
	}
	a = press(t, a, "esc")
	if !slices.Equal(shown(a), []string{cart, catalog}) || !a.FilesPanel.HasSelection() {
		t.Fatalf("back in the Files panel the selection is still there: %v", shown(a))
	}
	if a = press(t, a, "esc"); !slices.Equal(shown(a), []string{catalog}) || a.FilesPanel.HasSelection() {
		t.Fatalf("Esc stops the selection, the diff on the cursor's row: %v", shown(a))
	}
	if a = press(t, a, "esc"); a.Notice != "ZZ or :q quits" {
		t.Fatalf("Esc with no selection says how to quit, not %q", a.Notice)
	}
}

func TestTheSelectionFollowsTheStackFilter(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "2", "j", "j", "v", "j", "j", "]", "p")
	if a.Side(context.FilesPanel).Index("file:"+basket) >= 0 || !slices.Equal(shown(a), []string{memory, search}) {
		t.Fatalf("on #21 the selection keeps the files #21 changed: %v", shown(a))
	}
	if a.FilesPanel.GrowFrom != "" {
		t.Fatal("with its first row gone the selection stops growing")
	}
	if a = press(t, a, "[", "p"); !slices.Equal(shown(a), []string{memory, search}) {
		t.Fatalf("back on All the selection holds what it kept: %v", shown(a))
	}
	b, _ := opened(t, stack, 100, 30)
	b = press(t, b, "]", "p", "2", "v", "G")
	if !slices.Equal(shown(b), []string{memory, search, catalog}) {
		t.Fatalf("a selection under the filter holds only what #21 shows, and the context: %v", shown(b))
	}
}

func TestAJumpIntoAFileMovesTheFilesPanelsCursorToIt(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "2", "j", "z", "a")
	if a.Side(context.FilesPanel).Index("file:"+basket) >= 0 {
		t.Fatal("za folds the Basket folder")
	}
	a = typed(t, press(t, a, "space", "f"), "Basket/Basket")
	if got := selectedKey(a, context.FilesPanel); got != "file:"+basket || !slices.Equal(shown(a), []string{basket}) {
		t.Fatalf("the finder opens the file alone and unfolds the Files panel to it: on %s, %v", got, shown(a))
	}
	a = press(t, a, "esc", "3")
	if got := selectedKey(a, context.FilesPanel); got != "file:"+memory || !slices.Equal(shown(a), []string{memory}) {
		t.Fatalf("the Notes panel's first comment opens its file alone: on %s, %v", got, shown(a))
	}
	if a = press(t, a, "j"); selectedKey(a, context.FilesPanel) != "file:"+basket || !slices.Equal(shown(a), []string{basket}) {
		t.Fatalf("the next comment opens its own file: %v", shown(a))
	}
}

func TestTheDiagramOpensAComponentAndTheFilesPanelChoosesIt(t *testing.T) {
	a, _ := opened(t, stack, 100, 30)
	a = press(t, a, "space", "m", "space", "d")
	if !slices.Equal(a.FilesPanel.Chosen, []string{"component:root"}) || !slices.Equal(shown(a), []string{basket, memory, search, cart}) {
		t.Fatalf("the component's box opens its diff, its row chosen: %v %v", a.FilesPanel.Chosen, shown(a))
	}
	if a = press(t, a, "]", "f", "]", "f"); selectedKey(a, context.FilesPanel) != "component:root" {
		t.Fatalf("the Files panel's cursor stays on the component as the diff moves through it, not %s", selectedKey(a, context.FilesPanel))
	}
	a = press(t, a, "space", "m", "enter", "w", "space", "d")
	if !slices.Equal(a.FilesPanel.Chosen, []string{"component:root"}) || cursorAt(a) == basket+" file" || selectedKey(a, context.FilesPanel) != "component:root" {
		t.Fatalf("a type's box opens its component at the type, the Files panel on its row: %v at %s", a.FilesPanel.Chosen, cursorAt(a))
	}
}

func TestTheSelectKeyComesFromTheConfig(t *testing.T) {
	cmn := handedOver(t, map[string]map[string][]string{"filesPanel": {"select": {"V"}}})
	a, _ := openedIn(t, cmn, &changes{answers: map[string]answer{}}, stack, 100, 30)
	if a = press(t, a, "2", "v"); a.FilesPanel.GrowFrom != "" {
		t.Fatal("v no longer selects once the config gives select another key")
	}
	if a = press(t, a, "V", "j"); !slices.Equal(shown(a), []string{basket, memory, search, cart}) || a.FilesPanel.GrowFrom == "" {
		t.Fatalf("V selects instead: %v", shown(a))
	}
	none := handedOver(t, map[string]map[string][]string{"filesPanel": {"select": {}}})
	b, _ := openedIn(t, none, &changes{answers: map[string]answer{}}, stack, 100, 30)
	if b = press(t, b, "2", "v"); b.FilesPanel.GrowFrom != "" || strings.Contains(barSays(b), "Select") {
		t.Fatalf("with select disabled, v does nothing and the bar offers no selecting: %q", barSays(b))
	}
}
