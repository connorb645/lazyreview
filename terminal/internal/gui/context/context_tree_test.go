package context

import (
	"slices"
	"testing"
)

func TestTheFocusStackHoldsThePanelDrivingTheMainPanelThenTheMainPanelThenHome(t *testing.T) {
	tree := NewContextTree("github.com/octo/widgets/21")
	if got := tree.Stack(); !slices.Equal(got, []ContextKey{ReviewPanelKey}) || tree.Focus() != ReviewPanel || tree.Driving() != ReviewPanel {
		t.Fatalf("a review opens on the Review panel: %v", got)
	}
	tree = tree.Focused(MainPanel, FilesPanel).Showing(ScreenDiff)
	if got := tree.Stack(); !slices.Equal(got, []ContextKey{FilesPanelKey, DiffKey}) || tree.Focus() != MainPanel || tree.Driving() != FilesPanel {
		t.Fatalf("the diff with the keys, driven by Files: %v", got)
	}
	tree = tree.Showing(ScreenHome)
	if got := tree.Stack(); !slices.Equal(got, []ContextKey{FilesPanelKey, HomeMainKey, HomeKey}) || tree.Focus() != MainPanel {
		t.Fatalf("the home screen keeps the review's focus below it: %v", got)
	}
	tree = tree.Showing(ScreenDiagram).Focused(NotesPanel, DiagramPanel)
	if got := tree.Stack(); !slices.Equal(got, []ContextKey{NotesPanelKey}) || tree.Driving() != NotesPanel {
		t.Fatalf("a panel on the left with the keys drives the main panel: %v", got)
	}
	if NewContextTree("").Screen() != ScreenHome || !NewContextTree("").Has(HomeKey) {
		t.Fatal("with no review the app opens on the home screen")
	}
}

func TestPopupsSitOnTopInTheOrderTheyTakeKeys(t *testing.T) {
	tree := NewContextTree("github.com/octo/widgets/21").Opened(PromptKey).Opened(MenuKey).Opened(PostingKey)
	if got := tree.Stack(); !slices.Equal(got, []ContextKey{ReviewPanelKey, MenuKey, PostingKey, PromptKey}) {
		t.Fatalf("the prompt takes keys before posting, and posting before a menu: %v", got)
	}
	tree = tree.Opened(FinderKey).Focused(MainPanel, ReviewPanel)
	if got := tree.Stack(); !slices.Equal(got, []ContextKey{ReviewPanelKey, ReviewViewKey, FinderKey, PostingKey, PromptKey}) {
		t.Fatalf("the finder takes the menu's place, and popups stay above a change of focus: %v", got)
	}
	tree = tree.Closed(PromptKey).Closed(PostingKey).Closed(FinderKey)
	if got := tree.Stack(); !slices.Equal(got, []ContextKey{ReviewPanelKey, ReviewViewKey}) {
		t.Fatalf("closed popups leave the stack: %v", got)
	}
	for key, kind := range map[ContextKey]Kind{HomeKey: SideKind, FilesPanelKey: SideKind, DiffKey: MainKind, LabelsKey: PopupKind, GlobalKey: GlobalKind} {
		if key.Kind() != kind {
			t.Errorf("%s is of kind %d, not %d", key, key.Kind(), kind)
		}
	}
}
