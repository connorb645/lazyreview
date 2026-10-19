package context

import (
	"slices"

	"github.com/connorb645/lazyreview/terminal/internal/models"
)

// ContextTree is the gui's whole state: the global context, a context for
// each view, what the screen shows, and the focus stack, which says which
// context has the keys. At the bottom of the stack is the panel on the left
// that drives the main panel; then the main panel's context, when the main
// panel has the keys; then the home screen, while it shows; then any open
// popups, the one that takes keys first on top.
type ContextTree struct {
	GlobalContext
	Home         HomeContext
	ReviewPanel  SideContext
	FilesPanel   FilesContext
	NotesPanel   SideContext
	DiagramPanel SideContext
	ReviewView   ReviewViewContext
	Diff         DiffContext
	Diagram      DiagramContext
	Menu         MenuContext
	Posting      PostingContext
	Confirmation ConfirmationContext
	Prompt       PromptContext
	screen       Screen
	stack        []ContextKey
}

var (
	sideKeys   = [SideCount + 1]ContextKey{ReviewPanel: ReviewPanelKey, FilesPanel: FilesPanelKey, NotesPanel: NotesPanelKey, DiagramPanel: DiagramPanelKey}
	mainKeys   = map[Screen]ContextKey{ScreenHome: HomeMainKey, ScreenReview: ReviewViewKey, ScreenDiff: DiffKey, ScreenDiagram: DiagramKey}
	popupOrder = []ContextKey{MenuKey, LabelsKey, FinderKey, PostingKey, PostingConfirmationKey, ConfirmationKey, PromptKey}
)

// NewContextTree is the state of a gui that opens on a review, or with none
// on the home screen reading the open pull requests, the Review panel
// driving the main panel.
func NewContextTree(review string) ContextTree {
	t := ContextTree{screen: ScreenReview}
	t.Opening, t.Loading, t.Files = review, review != "", map[string]models.File{}
	t.ReviewView.Folded, t.FilesPanel.Folded = map[string]bool{}, map[string]bool{}
	if review == "" {
		t.screen, t.Home.Reading = ScreenHome, true
	}
	return t.based(ReviewPanel, ReviewPanel, nil)
}

// Screen is what the screen shows: the home screen, or a review's Review
// view, diff or diagram in the main panel.
func (t ContextTree) Screen() Screen {
	return t.screen
}

// Focus is the panel with the keys under any popup: the main panel or one
// on the left. On the home screen it is the one the review had.
func (t ContextTree) Focus() int {
	for _, key := range t.stack {
		if key.Kind() == MainKind {
			return MainPanel
		}
	}
	return t.Driving()
}

// Driving is the panel on the left whose place the main panel shows.
func (t ContextTree) Driving() int {
	for n := ReviewPanel; n <= SideCount; n++ {
		if sideKeys[n] == t.stack[0] {
			return n
		}
	}
	return ReviewPanel
}

// Showing is the tree with the screen showing s, the context of the main
// panel changing with it.
func (t ContextTree) Showing(s Screen) ContextTree {
	t.screen = s
	return t.based(t.Focus(), t.Driving(), t.popups())
}

// Focused is the tree with the keys in a panel, the main panel showing the
// place of the panel on the left driving it. A panel on the left with the
// keys drives the main panel itself.
func (t ContextTree) Focused(focus, driving int) ContextTree {
	if focus != MainPanel {
		driving = focus
	}
	return t.based(focus, driving, t.popups())
}

// Place is the context that reads keys in sequences: the one at the top of
// the focus stack under any popups.
func (t ContextTree) Place() ContextKey {
	for i := len(t.stack) - 1; i >= 0; i-- {
		if t.stack[i].Kind() != PopupKind {
			return t.stack[i]
		}
	}
	return GlobalKey
}

// Has says whether a context is on the focus stack.
func (t ContextTree) Has(key ContextKey) bool {
	return slices.Contains(t.stack, key)
}

// Stack is the focus stack, bottom first.
func (t ContextTree) Stack() []ContextKey {
	return slices.Clone(t.stack)
}

// Opened is the tree with a popup open on the focus stack, above those
// that take keys after it and below those that take keys before it. The
// menu and the finder share their popup, so each closes the other.
func (t ContextTree) Opened(key ContextKey) ContextTree {
	t = t.Closed(key)
	if key == MenuKey || key == FinderKey {
		t = t.Closed(MenuKey).Closed(FinderKey)
	}
	rank := slices.Index(popupOrder, key)
	at := len(t.stack)
	for at > 0 && t.stack[at-1].Kind() == PopupKind && slices.Index(popupOrder, t.stack[at-1]) > rank {
		at--
	}
	t.stack = slices.Insert(slices.Clone(t.stack), at, key)
	return t
}

// Closed is the tree with a popup no longer on the focus stack.
func (t ContextTree) Closed(key ContextKey) ContextTree {
	t.stack = slices.DeleteFunc(slices.Clone(t.stack), func(k ContextKey) bool { return k == key })
	return t
}

// MenuOpen says whether the menu, the keys or the finder is open.
func (t ContextTree) MenuOpen() bool {
	return t.Has(MenuKey) || t.Has(FinderKey)
}

// WithMenu is the tree with a menu open, as the finder when it is one.
func (t ContextTree) WithMenu(m MenuContext) ContextTree {
	key := MenuKey
	if m.Finding {
		key = FinderKey
	}
	t = t.Opened(key)
	t.Menu = m
	return t
}

// WithoutMenu is the tree with no menu or finder open.
func (t ContextTree) WithoutMenu() ContextTree {
	return t.Closed(MenuKey).Closed(FinderKey)
}

// WithoutPosting is the tree with posting and its confirmation closed.
func (t ContextTree) WithoutPosting() ContextTree {
	return t.Closed(PostingConfirmationKey).Closed(PostingKey)
}

// Popped says whether a popup is drawn over the panels, so none of them
// shows as having the keys.
func (t ContextTree) Popped() bool {
	return t.MenuOpen() || t.Has(PostingKey) || t.Has(ConfirmationKey)
}

// Said is the tree with a notice in the bar, bad or not.
func (t ContextTree) Said(notice string, bad bool) ContextTree {
	t.Notice, t.Bad = notice, bad
	return t
}

// Side is the context of a panel on the left, by its number.
func (t ContextTree) Side(n int) SideContext {
	switch n {
	case FilesPanel:
		return t.FilesPanel.SideContext
	case NotesPanel:
		return t.NotesPanel
	case DiagramPanel:
		return t.DiagramPanel
	}
	return t.ReviewPanel
}

// WithSide is the tree with the context of a panel on the left replaced.
func (t ContextTree) WithSide(n int, s SideContext) ContextTree {
	switch n {
	case FilesPanel:
		t.FilesPanel.SideContext = s
	case NotesPanel:
		t.NotesPanel = s
	case DiagramPanel:
		t.DiagramPanel = s
	default:
		t.ReviewPanel = s
	}
	return t
}

func (t ContextTree) popups() []ContextKey {
	var out []ContextKey
	for _, key := range t.stack {
		if key.Kind() == PopupKind {
			out = append(out, key)
		}
	}
	return out
}

func (t ContextTree) based(focus, driving int, popups []ContextKey) ContextTree {
	stack := []ContextKey{sideKeys[driving]}
	if focus == MainPanel {
		stack = append(stack, mainKeys[t.screen])
	}
	if t.screen == ScreenHome {
		stack = append(stack, HomeKey)
	}
	t.stack = append(stack, popups...)
	return t
}
