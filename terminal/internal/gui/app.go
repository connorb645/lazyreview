// Package gui is the terminal app: the panels and the keys, drawn from what
// the lazyreview server answers. It decides nothing the web app also needs.
package gui

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/connorb645/lazyreview/terminal/internal/commands/oscommands"
	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/common"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/keys"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const doubleClick = 400 * time.Millisecond

// App is the terminal app: the tools it was given and its state, the
// context tree. Each key press is Update, which returns the next App.
type App struct {
	c      *common.Common
	opts   Options
	md     *presentation.Markdown
	styles map[string]lipgloss.Style
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
		opts.Listen = listen
	}
	a := App{c: cmn, opts: opts, styles: diagram.Styles(cmn.Theme), md: presentation.NewMarkdown(cmn.Theme.Dark), ContextTree: context.NewContextTree(opts.Review)}
	a.Home = a.Home.Rebuilt(cmn.Tr, cmn.Theme)
	return a
}

// Init asks the server for the review to open, or for the home screen's
// open pull requests.
func (a App) Init() tea.Cmd {
	if a.Opening == "" {
		return readPulls(a.opts.Server)
	}
	return loadReview(a.c.Tr, a.opts.Server, a.Opening)
}

// Update takes one message, such as a key, a click, or an answer from the
// server, and returns the app as it then stands.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.Width, a.Height = msg.Width, msg.Height
		return a.relaid(), nil
	case types.PullsRead:
		return a.afterPulls(msg)
	case types.PullsLoaded:
		return a.afterLoaded(msg)
	case types.ReviewLoaded:
		if msg.ID != a.Opening {
			return a, nil
		}
		a.Loading = false
		if msg.Err != nil {
			a.Failed = msg.Err.Error()
			return a.say(msg.Err.Error(), true), nil
		}
		a.Review, a.Failed = &msg.Data, ""
		if a.Screen() != context.ScreenHome {
			a.ContextTree = a.Showing(context.ScreenReview)
		}
		a.Files, a.Diff, a.Filter, a.GrowFrom = map[string]server.File{}, context.DiffContext{}, nil, 0
		a.Diagram = context.NewDiagramContext(a.c.Tr, context.LevelAt(a.c.Tr, msg.Data.Page, types.DiagramPlace{Level: "components"}), "")
		a.ContextTree = a.Closed(context.LabelsKey)
		var listening tea.Cmd
		a, listening = a.listenTo(msg.Data.ID)
		return a.relaid(), listening
	case types.FilesLoaded:
		a.Loading = false
		if msg.Err != nil {
			a.ContextTree = a.Showing(a.CameFrom)
			return a.say(msg.Err.Error(), true).relaid(), nil
		}
		files := map[string]server.File{}
		for k, v := range a.Files {
			files[k] = v
		}
		for k, v := range msg.Files {
			files[k] = v
		}
		a.Files = files
		a.Diff.Files = files
		if a.Focus() != context.MainPanel {
			return a.followed()
		}
		next, cmd := a.openPlace(msg.Component, msg.Place)
		return next.(App).mirrored(), cmd
	case types.Edited:
		return a.afterEditing(msg)
	case types.Changed:
		return a.afterChange(msg)
	case types.Checked:
		return a.afterCheck(msg)
	case types.WhereShown:
		a.Where = msg
		return a, nil
	case types.Posted:
		return a.afterPost(msg), nil
	case types.StateLoaded:
		if msg.Err != nil {
			return a.say(a.c.Tr.CouldNotReadReviewAgain+": "+msg.Err.Error(), true), nil
		}
		return a.withState(msg.State), nil
	case types.LiveNotice:
		return a.afterNotice(msg)
	case types.LiveStopped:
		if a.Review != nil && msg.ID == a.Review.ID {
			a.LiveFor = ""
			return a.say(utils.ResolvePlaceholderString(a.c.Tr.StoppedHearing, map[string]string{"error": msg.Err.Error()}), true), nil
		}
		return a, nil
	case types.ReviewReloaded:
		return a.afterReload(msg)
	case tea.MouseMsg:
		next, cmd := a.mouse(msg)
		return next.(App).growingInReview(), cmd
	case tea.KeyMsg:
		next, cmd := a.key(msg)
		return next.(App).growingInReview(), cmd
	}
	if a.Has(context.PromptKey) {
		var cmd tea.Cmd
		a.Prompt.Input, cmd = a.Prompt.Input.Update(msg)
		return a, cmd
	}
	return a, nil
}

func (a App) key(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.Has(context.PromptKey) {
		return a.promptKey(msg)
	}
	name := keyName(msg)
	if a.Has(context.ConfirmationKey) {
		return a.confirmKey(name)
	}
	if a.Has(context.PostingKey) {
		return a.postingKey(name)
	}
	if a.menuOpen() && a.Menu.Finding {
		return a.finderKey(msg)
	}
	if a.Screen() == context.ScreenDiagram && a.Has(context.LabelsKey) {
		return a.labelKey(name).mirrored(), nil
	}
	if a.menuOpen() {
		switch name {
		case "enter":
			return a.openRow()
		case "j", "down":
			a.Menu = a.Menu.Move(1, a.menuRows())
			return a, nil
		case "k", "up":
			a.Menu = a.Menu.Move(-1, a.menuRows())
			return a, nil
		case "q", "esc":
			return a.withoutMenu(), nil
		}
		a = a.withoutMenu()
	}
	var res keys.Result
	a.Keys, res = a.Keys.Read(name, a.bindings())
	if !res.Waiting {
		a.Notice, a.Bad = "", false
	}
	if res.Action == "" {
		return a, nil
	}
	next, cmd := a.do(res.Action, res.Count, res.Counted)
	if after, ok := next.(App); ok {
		after = after.mirrored()
		if where := after.whereNow(); where != nil {
			return after, tea.Batch(cmd, where)
		}
		return after, cmd
	}
	return next, cmd
}

func (a App) bindings() keys.Bindings {
	if a.Screen() == context.ScreenHome {
		return homeKeys
	}
	if a.Focus() != context.MainPanel {
		return everyView.With(sideKeys).With(sidePanelKeys[a.Focus()])
	}
	switch a.Screen() {
	case context.ScreenDiff:
		return everyView.With(diffKeys)
	case context.ScreenDiagram:
		return everyView.With(diagramKeys)
	}
	return everyView.With(reviewKeys)
}

func (a App) do(action string, count int, counted bool) (tea.Model, tea.Cmd) {
	h := a.mainHeight()
	switch action {
	case "quit":
		return a, tea.Quit
	case "refresh":
		return a.refresh()
	case "command", "find":
		return a.ask(map[string]string{"command": ":", "find": "/"}[action]), nil
	case "keys":
		return a.showKeys(), nil
	case "close":
		return a.closed(), nil
	case "focus main", "focus review", "focus files", "focus notes", "focus diagram":
		return a.focusOn(map[string]int{"focus main": context.MainPanel, "focus review": context.ReviewPanel, "focus files": context.FilesPanel, "focus notes": context.NotesPanel, "focus diagram": context.DiagramPanel}[action])
	case "bigger", "smaller":
		by := 1
		if action == "smaller" {
			by = int(context.ScreenModes) - 1
		}
		a.Mode = (a.Mode + context.ScreenMode(by)) % context.ScreenModes
		return a.relaid(), nil
	case "review":
		switch {
		case a.Review == nil && a.Screen() == context.ScreenHome:
			return a.say(a.c.Tr.NoReviewOpenYet, true), nil
		case a.Review == nil:
			return a.say(a.c.Tr.ReviewNotOpenYet, true), nil
		case a.Screen() != context.ScreenHome:
			a = a.leave()
		}
		a.ContextTree = a.Showing(context.ScreenReview).Focused(context.MainPanel, context.ReviewPanel)
		return a.relaid(), nil
	case "home":
		return a.goHome()
	case "diagram":
		return a.openDiagram(), nil
	case "finder":
		return a.openFinder(), nil
	case "next pull":
		return a.filterBy(count), nil
	case "previous pull":
		return a.filterBy(-count), nil
	case "jump back":
		return a.jumpBy(true)
	case "jump forward":
		return a.jumpBy(false)
	case "github":
		return a.openGitHub()
	case "web":
		return a.openWeb()
	}
	if a.Screen() == context.ScreenHome {
		return a.homeAction(action, count)
	}
	if a.Review == nil {
		return a, nil
	}
	if a.Focus() != context.MainPanel {
		return a.sideAction(action, count)
	}
	switch a.Screen() {
	case context.ScreenDiff:
		return a.diffAction(action, count, counted, h)
	case context.ScreenDiagram:
		return a.diagramAction(action, count)
	}
	return a.reviewAction(action, count, h)
}

func (a App) closed() App {
	switch {
	case a.menuOpen():
		a = a.withoutMenu()
	case a.Screen() == context.ScreenHome:
		return a.say(a.c.Tr.HowToQuit, false)
	case a.Focus() == context.ReviewPanel && len(a.Filter) > 0:
		return a.setFilter(nil)
	case a.Screen() == context.ScreenDiff && a.Diff.Selecting:
		a.Diff.Selecting = false
	case a.Focus() == context.MainPanel && a.Review != nil:
		a.ContextTree = a.Focused(a.Driving(), a.Driving())
		return a.relaid()
	default:
		return a.say(a.c.Tr.HowToQuit, false)
	}
	return a
}

func (a App) reviewAction(action string, count, h int) (tea.Model, tea.Cmd) {
	v := a.ReviewView.View
	switch action {
	case "down":
		a.ReviewView.View = v.Move(count, h)
	case "up":
		a.ReviewView.View = v.Move(-count, h)
	case "top":
		a.ReviewView.View = v.To(0, h)
	case "bottom":
		a.ReviewView.View = v.To(len(v.Items)-1, h)
	case "next section", "previous section":
		a.ReviewView = a.ReviewView.Section(action == "previous section", count, h)
	case "open":
		if v.Cursor >= len(v.Items) {
			return a, nil
		}
		it := v.Items[v.Cursor]
		switch {
		case it.Kind == "section":
			return a.reviewAction("fold", 1, h)
		case it.Kind == "component":
			a = a.leave()
			a.ContextTree = a.Focused(a.Focus(), context.FilesPanel)
			return a.openPlace(it.Component, nil)
		case it.Place != nil:
			a = a.leave()
			a.ContextTree = a.Focused(a.Focus(), context.FilesPanel)
			if it.Kind == "comment" || it.Kind == "thread" || it.Kind == "conversation" {
				a.ContextTree = a.Focused(a.Focus(), context.NotesPanel)
			}
			return a.openPlace(a.componentOf(it.Place.Path), it.Place)
		}
	case "edit":
		return a.editQueued()
	case "drop":
		return a.dropQueued(), nil
	case "post":
		pull := 0
		if v.Cursor < len(v.Items) {
			pull = v.Items[v.Cursor].Pull
		}
		return a.openPosting(pull), nil
	case "fold":
		id := a.ReviewView.SectionAt()
		if id == "" {
			return a, nil
		}
		folded := map[string]bool{}
		for k, val := range a.ReviewView.Folded {
			folded[k] = val
		}
		folded[id] = !folded[id]
		a.ReviewView.Folded = folded
		for i, it := range a.ReviewView.View.Items {
			if it.Kind == "section" && it.Section == id {
				a.ReviewView.View.Cursor = i
			}
		}
		return a.relaid(), nil
	}
	return a, nil
}

func (a App) openGitHub() (tea.Model, tea.Cmd) {
	address := ""
	switch {
	case a.Screen() == context.ScreenHome:
		if _, _, m, ok := a.Home.Pull(); ok {
			address = m.URL
		}
	case a.Review == nil:
	case a.Focus() != context.MainPanel:
		if row, ok := a.Side(a.Focus()).Selected(); ok {
			address = row.Item.URL
		}
	case a.Screen() == context.ScreenDiff:
		if n, _, ok := a.noteAt(); ok && n.Kind == "thread" && len(n.Thread.Comments) > 0 {
			address = n.Thread.Comments[0].URL
		}
	case a.Screen() == context.ScreenReview:
		if v := a.ReviewView.View; v.Cursor < len(v.Items) {
			address = v.Items[v.Cursor].URL
		}
	}
	if address == "" && a.Screen() != context.ScreenHome && a.Review != nil && len(a.Review.State.Members) > 0 {
		address = a.Review.State.Members[0].URL
	}
	if address == "" {
		return a.say(a.c.Tr.NoPullRequestOnGitHub, true), nil
	}
	return a.opened(address)
}

func (a App) openWeb() (tea.Model, tea.Cmd) {
	base, err := a.opts.Server.Address()
	if err != nil {
		return a.say(err.Error(), true), nil
	}
	if a.Screen() != context.ScreenReview || a.Review == nil {
		return a.opened(base + "/")
	}
	parts := strings.Split(a.Review.ID, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return a.opened(base + "/review/" + strings.Join(parts, "/") + "#review")
}

func (a App) opened(address string) (tea.Model, tea.Cmd) {
	if err := a.opts.Open(address); err != nil {
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.CouldNotOpen, map[string]string{"name": address})+": "+err.Error(), true), nil
	}
	return a.say(utils.ResolvePlaceholderString(a.c.Tr.Opened, map[string]string{"address": address}), false), nil
}

func (a App) showKeys() App {
	tr := a.c.Tr
	rows := map[context.Screen][]presentation.HelpRow{context.ScreenReview: reviewHelp(tr), context.ScreenDiff: diffHelp(tr), context.ScreenDiagram: diagramHelp(tr)}[a.Screen()]
	if a.Focus() != context.MainPanel {
		rows = sideHelp(tr)[a.Focus()]
	}
	heading, every, everyRows := a.panelName(), tr.EveryPanel, everyViewHelp(tr)
	if a.Screen() == context.ScreenHome {
		heading, rows, every, everyRows = tr.HomeTitle, homeHelp(tr), tr.Marks, homeMarks(tr)
	}
	pn := context.MenuContext{Menu: presentation.Menu{Title: tr.KeysTitle}}
	pn.Rows = append(pn.Rows, presentation.MenuRow{Heading: heading})
	for _, r := range rows {
		pn.Rows = append(pn.Rows, presentation.MenuRow{Left: r.Keys, Right: r.Words})
	}
	pn.Rows = append(pn.Rows, presentation.MenuRow{Heading: every})
	for _, r := range everyRows {
		pn.Rows = append(pn.Rows, presentation.MenuRow{Left: r.Keys, Right: r.Words})
	}
	return a.withMenu(pn.First())
}

func (a App) ask(prefix string) App {
	in := textinput.New()
	in.Prompt = prefix
	in.Cursor.SetMode(cursor.CursorStatic)
	in.Focus()
	a.ContextTree = a.Opened(context.PromptKey)
	a.Prompt = context.PromptContext{Input: in, Kind: prefix}
	return a
}

func (a App) promptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		a.ContextTree = a.Closed(context.PromptKey)
		return a, nil
	case tea.KeyEnter:
		typed := strings.TrimSpace(a.Prompt.Input.Value())
		kind := a.Prompt.Kind
		a.ContextTree = a.Closed(context.PromptKey)
		switch {
		case a.Screen() == context.ScreenDiagram && (kind == "/" || kind == "?"):
			a.Diagram.Query = typed
			next, cmd := a.findBox(a.SearchBack)
			return next.(App).mirrored(), cmd
		case kind == "/" || kind == "?":
			a.Diff.Query = typed
			next, cmd := a.matchNext(a.SearchBack)
			return next.(App).mirrored(), cmd
		}
		return a.command(typed)
	}
	var cmd tea.Cmd
	a.Prompt.Input, cmd = a.Prompt.Input.Update(msg)
	return a, cmd
}

func (a App) command(typed string) (tea.Model, tea.Cmd) {
	switch typed {
	case "":
		return a, nil
	case "q", "q!", "quit", "qa", "qa!":
		return a, tea.Quit
	case "refresh":
		return a.refresh()
	}
	if n, err := strconv.Atoi(typed); err == nil && a.Screen() == context.ScreenDiff && a.Review != nil {
		a = a.leave()
		a.ContextTree = a.Focused(context.MainPanel, a.Driving())
		next, cmd := a.toLine(n)
		return next.(App).mirrored(), cmd
	}
	if arg, ok := strings.CutPrefix(typed, "pull "); ok {
		return a.pullCommand(strings.TrimSpace(arg)), nil
	}
	return a.say(utils.ResolvePlaceholderString(a.c.Tr.NotACommand, map[string]string{"command": typed}), true), nil
}

func (a App) relaid() App {
	if a.Screen() == context.ScreenHome {
		return a.homeRelaid()
	}
	if a.Width == 0 || a.Review == nil {
		return a
	}
	a = a.sidesBuilt()
	h := a.mainHeight()
	switch {
	case a.Screen() == context.ScreenDiagram:
		a = a.diagramShown()
	case a.Screen() == context.ScreenDiff && a.Diff.Files != nil:
		a.Diff.Width = a.mainWidth()
		a.Diff = a.Diff.Rebuilt(a.c.Tr, a.md, a.opts.Location, *a.Review, a.Filter, h)
	}
	a.ReviewView = a.ReviewView.Rebuilt(a.c.Tr, a.c.Theme, a.md, a.opts.Location, *a.Review, a.Filter, a.mainWidth(), h)
	return a.mirrored()
}

func (a App) refresh() (tea.Model, tea.Cmd) {
	if a.Screen() == context.ScreenHome {
		return a.readHome()
	}
	if a.Review == nil {
		a.Loading, a.Failed = true, ""
		return a.say(a.c.Tr.OpeningReviewAgain, false), loadReview(a.c.Tr, a.opts.Server, a.Opening)
	}
	cmd := reloadReview(a.c.Tr, a.opts.Server, a.Review.ID, true)
	if a.LiveFor == "" {
		var listening tea.Cmd
		a, listening = a.listenTo(a.Review.ID)
		cmd = tea.Batch(cmd, listening)
	}
	return a.say(a.c.Tr.LoadingReviewAgain, false), cmd
}

func (a App) say(notice string, bad bool) App {
	a.Notice, a.Bad = notice, bad
	return a
}

func (a App) layout() layout {
	var wants [context.SideCount + 1]int
	for n := context.ReviewPanel; n <= context.SideCount; n++ {
		wants[n] = max(1, len(a.Side(n).Rows))
	}
	return lay(a.Width, a.Height, a.Focus(), a.Driving(), a.Mode, wants)
}

func (a App) mainRect() views.Rect {
	if r := a.layout().main; r.Shown() {
		return r.Inner()
	}
	a.ContextTree = a.Focused(context.MainPanel, a.Driving())
	return a.layout().main.Inner()
}

func (a App) mainHeight() int {
	return max(1, a.mainRect().H)
}

func (a App) mainWidth() int {
	return max(1, a.mainRect().W)
}

func (a App) sideHeight(n int) int {
	return max(1, a.layout().sides[n].Inner().H)
}

func (a App) menuRows() int {
	return max(1, min(a.Height*2/3, 24)-2)
}

func loadReview(tr *i18n.TranslationSet, c *server.Client, id string) tea.Cmd {
	couldNotOpen := utils.ResolvePlaceholderString(tr.CouldNotOpen, map[string]string{"name": id})
	return func() tea.Msg {
		state, err := c.State(id)
		if err != nil {
			return types.ReviewLoaded{ID: id, Err: fmt.Errorf("%s: %w", couldNotOpen, err)}
		}
		page, err := c.Page(id)
		if err != nil {
			return types.ReviewLoaded{ID: id, Err: fmt.Errorf("%s: %w", couldNotOpen, err)}
		}
		return types.ReviewLoaded{ID: id, Data: types.ReviewData{ID: id, State: state, Page: page}}
	}
}

func keyName(msg tea.KeyMsg) string {
	switch msg.Type {
	case tea.KeySpace:
		return "space"
	case tea.KeyEnter:
		return "enter"
	case tea.KeyEsc:
		return "esc"
	case tea.KeyTab:
		return "tab"
	case tea.KeyUp:
		return "up"
	case tea.KeyDown:
		return "down"
	}
	return msg.String()
}
