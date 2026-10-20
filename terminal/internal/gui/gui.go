// Package gui is the terminal app: the panels and the keys, drawn from the
// reviews it is given.
package gui

import (
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/connorb645/lazyreview/terminal/internal/commands/oscommands"
	"github.com/connorb645/lazyreview/terminal/internal/common"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
)

// App is the terminal app: what its helpers share, the helpers, each
// context's controller, and its state, the context tree. Each key press is
// Update, which returns the next App.
type App struct {
	c           *helpers.HelperCommon
	helpers     *helpers.Helpers
	controllers *controllers.Controllers
	styles      map[string]lipgloss.Style
	context.ContextTree
}

// New returns the app, which opens on the review opts names, or with none on
// the home screen.
func New(cmn *common.Common, opts Options) App {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Location == nil {
		opts.Location = time.Local
	}
	if opts.Home == "" {
		opts.Home, _ = os.UserHomeDir()
	}
	if opts.Open == nil {
		opts.Open = oscommands.OpenInBrowser
	}
	if opts.Editor == "" {
		opts.Editor = "vi"
	}
	if opts.Exec == nil {
		opts.Exec = tea.ExecProcess
	}
	if opts.Listen == nil {
		opts.Listen = helpers.Listen
	}
	if opts.Tick == nil {
		opts.Tick = tea.Tick
	}
	hc := &helpers.HelperCommon{
		Common: cmn, Reviews: opts.Reviews, Open: opts.Open, Now: opts.Now, Location: opts.Location, Home: opts.Home,
		Editor: opts.Editor, Exec: opts.Exec, Listen: opts.Listen, Tick: opts.Tick, Markdown: presentation.NewMarkdown(cmn.Theme.Dark),
	}
	a := App{c: hc, helpers: helpers.NewHelpers(hc), controllers: controllers.NewControllers(hc), styles: diagram.Styles(cmn.Theme), ContextTree: context.NewContextTree(opts.Review)}
	a.Home = a.Home.Rebuilt(cmn.Tr, cmn.Theme)
	return a
}

// Init reads the review to open, or the home screen's open pull requests.
func (a App) Init() tea.Cmd {
	if a.Opening == "" {
		return a.helpers.Home.ReadPulls()
	}
	return a.helpers.Review.Load(a.Opening)
}

// Update takes one message, such as a key, a click, or a review read, and
// returns the app as it then stands.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	h, s := a.helpers, a.ContextTree
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.Width, s.Height = msg.Width, msg.Height
		s = h.Refresh.Relaid(s)
	case types.PullsRead:
		s, cmd = h.Home.AfterPulls(s, msg)
	case types.PullsLoaded:
		s, cmd = h.Home.AfterLoaded(s, msg)
	case types.HiddenChanged:
		s = h.Home.AfterHidden(s, msg)
	case types.ReviewLoaded:
		s, cmd = h.Review.AfterLoaded(s, msg)
	case types.FilesLoaded:
		s, cmd = h.Diff.AfterFiles(s, msg)
	case types.Edited:
		s, cmd = h.Editing.AfterEditing(s, msg)
	case types.Changed:
		s, cmd = h.Editing.AfterChange(s, msg)
	case types.Checked:
		s, cmd = h.Editing.AfterCheck(s, msg)
	case types.WhereShown:
		s.Where = msg
	case types.Posted:
		s = h.Posting.AfterPost(s, msg)
	case types.StateLoaded:
		s = h.Review.AfterState(s, msg)
	case types.LiveNotice:
		s, cmd = h.Live.AfterNotice(s, msg)
	case types.LiveStopped:
		s = h.Live.AfterStopped(s, msg)
	case types.ReviewReloaded:
		s, cmd = h.Review.AfterReload(s, msg)
	case types.Spun:
		s = h.AppStatus.AfterSpin(s)
	case tea.MouseMsg:
		s, cmd = a.mouse(s, msg)
		s = h.Files.GrowingInFiles(h.Filter.GrowingInReview(s))
	case tea.KeyMsg:
		s, cmd = a.key(s, msg)
		s = h.Files.GrowingInFiles(h.Filter.GrowingInReview(s))
	default:
		if s.Has(context.PromptKey) {
			s, cmd = h.Prompt.Typed(s, msg)
		}
	}
	s, spin := h.AppStatus.Kept(s)
	a.ContextTree = s
	return a, tea.Batch(cmd, spin)
}
