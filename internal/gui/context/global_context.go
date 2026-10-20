package context

import (
	stdcontext "context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/keys"
	"github.com/connorb645/lazyreview/internal/models"
)

// GlobalContext is the state no single view holds: the screen's size and
// mode, the review open or opening, its files loaded, the pull request
// filter and its highlight, the jumps, the notice in the bar, the keys typed
// toward a sequence, the last click, the drafts and the editing under way,
// the questions sent to Claude and not yet answered, the conversation last
// stopped here, the spinner's frame and whether its next tick is due, and
// the stream of the review's changes.
type GlobalContext struct {
	Width      int
	Height     int
	Mode       ScreenMode
	Opening    string
	Review     *types.ReviewData
	Failed     string
	Loading    bool
	Files      map[string]models.File
	Filter     types.PullFilter
	GrowFrom   int
	CameFrom   Screen
	Jumps      []Jump
	JumpAt     int
	SearchBack bool
	Keys       keys.Reader
	Notice     string
	Bad        bool
	ClickedAt  time.Time
	ClickedY   int
	Drafts     map[string]string
	Where      types.WhereShown
	Choosing   *types.Editing
	EditingNow bool
	Asking     int
	Stopped    int
	Spin       int
	Spinning   bool
	Live       <-chan tea.Msg
	StopLive   stdcontext.CancelFunc
	LiveFor    string
}

// ShownPulls are the pull requests the screen shows: the filter, or none,
// the whole stack, when it holds every pull request of the stack.
func (g GlobalContext) ShownPulls() types.PullFilter {
	if g.Review == nil {
		return g.Filter
	}
	return g.Filter.Over(presentation.StackPulls(*g.Review))
}
