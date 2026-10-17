package context

import (
	stdcontext "context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/keys"
)

// GlobalContext is the state no single view holds: the screen's size and
// mode, the review open or opening, its files loaded, the pull request
// filter and its highlight, the jumps, the notice in the bar, the keys typed
// toward a sequence, the last click, the drafts and the editing under way,
// and the stream of the review's changes.
type GlobalContext struct {
	Width      int
	Height     int
	Mode       ScreenMode
	Opening    string
	Review     *types.ReviewData
	Failed     string
	Loading    bool
	Files      map[string]server.File
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
	Live       <-chan tea.Msg
	StopLive   stdcontext.CancelFunc
	LiveFor    string
}
