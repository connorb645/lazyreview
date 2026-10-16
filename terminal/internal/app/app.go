// Package app is the terminal app: the panels and the keys, drawn from what
// the lazyreview server answers. It decides nothing the web app also needs.
package app

import (
	"context"
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

	"github.com/connorb645/lazyreview/terminal/internal/keys"
	"github.com/connorb645/lazyreview/terminal/internal/server"
)

const doubleClick = 400 * time.Millisecond

// App is the terminal app's state: the home screen of open pull requests,
// the review open, if any, what the main panel shows, which panel has the
// keys, what the server gave it, where each cursor is, and what has been
// typed toward a key sequence. Each key press is Update, which returns the
// next App.
type App struct {
	opts       Options
	p          palette
	md         *markdown
	width      int
	height     int
	screen     screen
	home       homeView
	opening    string
	focus      int
	driving    int
	sides      [sideCount + 1]sidePanel
	tree       map[string]bool
	mode       screenMode
	review     *reviewData
	failed     string
	rv         reviewView
	diff       diffView
	dg         diagramView
	styles     map[string]lipgloss.Style
	files      map[string]server.File
	filter     int
	cameFrom   screen
	jumps      []jump
	jumpAt     int
	searchBack bool
	keys       keys.Reader
	menu       *menu
	prompt     textinput.Model
	prompting  string
	notice     string
	bad        bool
	loading    bool
	clickedAt  time.Time
	clickedY   int
	drafts     map[string]string
	where      whereShown
	confirming *confirmation
	choosing   *editing
	posting    *posting
	editingNow bool
	live       <-chan tea.Msg
	stopLive   context.CancelFunc
	liveFor    string
}

// New returns the app, which opens on the review opts names, or with none on
// the home screen.
func New(opts Options) App {
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
		opts.Open = openInBrowser
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
	p := newPalette(opts.Dark)
	a := App{opts: opts, p: p, styles: diagramStyles(p), md: newMarkdown(opts.Dark), loading: opts.Review != "", screen: screenReview, opening: opts.Review,
		focus: reviewPanel, driving: reviewPanel, rv: reviewView{folded: map[string]bool{}}, files: map[string]server.File{}, tree: map[string]bool{}}
	if opts.Review == "" {
		a.screen, a.home.reading = screenHome, true
	}
	a.home = a.home.rebuilt(p)
	return a
}

// Init asks the server for the review to open, or for the home screen's
// open pull requests.
func (a App) Init() tea.Cmd {
	if a.opening == "" {
		return readPulls(a.opts.Server)
	}
	return loadReview(a.opts.Server, a.opening)
}

// Update takes one message, such as a key, a click, or an answer from the
// server, and returns the app as it then stands.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a.relaid(), nil
	case pullsRead:
		return a.afterPulls(msg)
	case pullsLoaded:
		return a.afterLoaded(msg)
	case reviewLoaded:
		if msg.id != a.opening {
			return a, nil
		}
		a.loading = false
		if msg.err != nil {
			a.failed = msg.err.Error()
			return a.say(msg.err.Error(), true), nil
		}
		a.review, a.failed = &msg.data, ""
		if a.screen != screenHome {
			a.screen = screenReview
		}
		a.files, a.diff, a.filter = map[string]server.File{}, diffView{}, 0
		a.dg = newDiagramView(levelAt(msg.data.page, diagramPlace{level: "components"}), "")
		var listening tea.Cmd
		a, listening = a.listenTo(msg.data.id)
		return a.relaid(), listening
	case filesLoaded:
		a.loading = false
		if msg.err != nil {
			a.screen = a.cameFrom
			return a.say(msg.err.Error(), true).relaid(), nil
		}
		files := map[string]server.File{}
		for k, v := range a.files {
			files[k] = v
		}
		for k, v := range msg.files {
			files[k] = v
		}
		a.files = files
		a.diff.files = files
		if a.focus != mainPanel {
			return a.followed()
		}
		next, cmd := a.openPlace(msg.component, msg.place)
		return next.(App).mirrored(), cmd
	case edited:
		return a.afterEditing(msg)
	case changed:
		return a.afterChange(msg)
	case checked:
		return a.afterCheck(msg)
	case whereShown:
		a.where = msg
		return a, nil
	case posted:
		return a.afterPost(msg), nil
	case stateLoaded:
		if msg.err != nil {
			return a.say("could not read the review again: "+msg.err.Error(), true), nil
		}
		return a.withState(msg.state), nil
	case liveNotice:
		return a.afterNotice(msg)
	case liveStopped:
		if a.review != nil && msg.id == a.review.id {
			a.liveFor = ""
			return a.say("Stopped hearing changes made elsewhere: "+msg.err.Error()+". R loads the review again.", true), nil
		}
		return a, nil
	case reviewReloaded:
		return a.afterReload(msg)
	case tea.MouseMsg:
		return a.mouse(msg)
	case tea.KeyMsg:
		return a.key(msg)
	}
	if a.prompting != "" {
		var cmd tea.Cmd
		a.prompt, cmd = a.prompt.Update(msg)
		return a, cmd
	}
	return a, nil
}

func (a App) key(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.prompting != "" {
		return a.promptKey(msg)
	}
	name := keyName(msg)
	if a.confirming != nil {
		return a.confirmKey(name)
	}
	if a.posting != nil {
		return a.postingKey(name)
	}
	if a.menu != nil && a.menu.finding {
		return a.finderKey(msg)
	}
	if a.screen == screenDiagram && a.dg.labels != nil {
		return a.labelKey(name).mirrored(), nil
	}
	if a.menu != nil {
		switch name {
		case "enter":
			return a.openRow()
		case "j", "down":
			moved := a.menu.move(1, a.menuRows())
			a.menu = &moved
			return a, nil
		case "k", "up":
			moved := a.menu.move(-1, a.menuRows())
			a.menu = &moved
			return a, nil
		case "q", "esc":
			a.menu = nil
			return a, nil
		}
		a.menu = nil
	}
	var res keys.Result
	a.keys, res = a.keys.Read(name, a.bindings())
	if !res.Waiting {
		a.notice, a.bad = "", false
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
	if a.screen == screenHome {
		return homeKeys
	}
	if a.focus != mainPanel {
		return everyView.With(sideKeys).With(sidePanelKeys[a.focus])
	}
	switch a.screen {
	case screenDiff:
		return everyView.With(diffKeys)
	case screenDiagram:
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
		return a.focusOn(map[string]int{"focus main": mainPanel, "focus review": reviewPanel, "focus files": filesPanel, "focus notes": notesPanel, "focus diagram": diagramPanel}[action])
	case "bigger", "smaller":
		by := 1
		if action == "smaller" {
			by = int(screenModes) - 1
		}
		a.mode = (a.mode + screenMode(by)) % screenModes
		return a.relaid(), nil
	case "review":
		switch {
		case a.review == nil && a.screen == screenHome:
			return a.say("No review is open yet: Enter opens one", true), nil
		case a.review == nil:
			return a.say("the review is not open yet; R tries again", true), nil
		case a.screen != screenHome:
			a = a.leave()
		}
		a.screen, a.focus, a.driving = screenReview, mainPanel, reviewPanel
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
	if a.screen == screenHome {
		return a.homeAction(action, count)
	}
	if a.review == nil {
		return a, nil
	}
	if a.focus != mainPanel {
		return a.sideAction(action, count)
	}
	switch a.screen {
	case screenDiff:
		return a.diffAction(action, count, counted, h)
	case screenDiagram:
		return a.diagramAction(action, count)
	}
	return a.reviewAction(action, count, h)
}

func (a App) closed() App {
	switch {
	case a.menu != nil:
		a.menu = nil
	case a.screen == screenHome:
		return a.say("ZZ or :q quits", false)
	case a.screen == screenDiff && a.diff.selecting:
		a.diff.selecting = false
	case a.focus == mainPanel && a.review != nil:
		a.focus = a.driving
		return a.relaid()
	default:
		return a.say("ZZ or :q quits", false)
	}
	return a
}

func (a App) reviewAction(action string, count, h int) (tea.Model, tea.Cmd) {
	v := a.rv.view
	switch action {
	case "down":
		a.rv.view = v.move(count, h)
	case "up":
		a.rv.view = v.move(-count, h)
	case "top":
		a.rv.view = v.to(0, h)
	case "bottom":
		a.rv.view = v.to(len(v.items)-1, h)
	case "next section", "previous section":
		a.rv = a.rv.section(action == "previous section", count, h)
	case "open":
		if v.cursor >= len(v.items) {
			return a, nil
		}
		it := v.items[v.cursor]
		switch {
		case it.kind == "section":
			return a.reviewAction("fold", 1, h)
		case it.kind == "component":
			a = a.leave()
			a.driving = filesPanel
			return a.openPlace(it.component, nil)
		case it.place != nil:
			a = a.leave()
			a.driving = filesPanel
			if it.kind == "comment" || it.kind == "thread" || it.kind == "conversation" {
				a.driving = notesPanel
			}
			return a.openPlace(a.componentOf(it.place.Path), it.place)
		}
	case "edit":
		return a.editQueued()
	case "drop":
		return a.dropQueued(), nil
	case "post":
		pull := 0
		if v.cursor < len(v.items) {
			pull = v.items[v.cursor].pull
		}
		return a.openPosting(pull), nil
	case "fold":
		id := a.rv.sectionAt()
		if id == "" {
			return a, nil
		}
		folded := map[string]bool{}
		for k, val := range a.rv.folded {
			folded[k] = val
		}
		folded[id] = !folded[id]
		a.rv.folded = folded
		for i, it := range a.rv.view.items {
			if it.kind == "section" && it.section == id {
				a.rv.view.cursor = i
			}
		}
		return a.relaid(), nil
	}
	return a, nil
}

func (a App) openGitHub() (tea.Model, tea.Cmd) {
	address := ""
	switch {
	case a.screen == screenHome:
		if _, _, m, ok := a.home.pull(); ok {
			address = m.URL
		}
	case a.review == nil:
	case a.focus != mainPanel:
		if row, ok := a.sides[a.focus].selected(); ok {
			address = row.it.url
		}
	case a.screen == screenDiff:
		if n, _, ok := a.noteAt(); ok && n.kind == "thread" && len(n.thread.Comments) > 0 {
			address = n.thread.Comments[0].URL
		}
	case a.screen == screenReview:
		if v := a.rv.view; v.cursor < len(v.items) {
			address = v.items[v.cursor].url
		}
	}
	if address == "" && a.screen != screenHome && a.review != nil && len(a.review.state.Members) > 0 {
		address = a.review.state.Members[0].URL
	}
	if address == "" {
		return a.say("this has no pull request on GitHub", true), nil
	}
	return a.opened(address)
}

func (a App) openWeb() (tea.Model, tea.Cmd) {
	base, err := a.opts.Server.Address()
	if err != nil {
		return a.say(err.Error(), true), nil
	}
	if a.screen != screenReview || a.review == nil {
		return a.opened(base + "/")
	}
	parts := strings.Split(a.review.id, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return a.opened(base + "/review/" + strings.Join(parts, "/") + "#review")
}

func (a App) opened(address string) (tea.Model, tea.Cmd) {
	if err := a.opts.Open(address); err != nil {
		return a.say("could not open "+address+": "+err.Error(), true), nil
	}
	return a.say("opened "+address, false), nil
}

func (a App) showKeys() App {
	rows := map[screen][]helpRow{screenReview: reviewHelp, screenDiff: diffHelp, screenDiagram: diagramHelp}[a.screen]
	if a.focus != mainPanel {
		rows = sideHelp[a.focus]
	}
	heading, every, everyRows := a.panelName(), "Every panel", everyViewHelp
	if a.screen == screenHome {
		heading, rows, every, everyRows = "Home", homeHelp, "Marks", homeMarks
	}
	pn := menu{title: "Keys"}
	pn.rows = append(pn.rows, menuRow{heading: heading})
	for _, r := range rows {
		pn.rows = append(pn.rows, menuRow{left: r.keys, right: r.words})
	}
	pn.rows = append(pn.rows, menuRow{heading: every})
	for _, r := range everyRows {
		pn.rows = append(pn.rows, menuRow{left: r.keys, right: r.words})
	}
	pn = pn.first()
	a.menu = &pn
	return a
}

func (a App) ask(prefix string) App {
	in := textinput.New()
	in.Prompt = prefix
	in.Cursor.SetMode(cursor.CursorStatic)
	in.Focus()
	a.prompt, a.prompting = in, prefix
	return a
}

func (a App) promptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		a.prompting = ""
		return a, nil
	case tea.KeyEnter:
		typed := strings.TrimSpace(a.prompt.Value())
		kind := a.prompting
		a.prompting = ""
		switch {
		case a.screen == screenDiagram && (kind == "/" || kind == "?"):
			a.dg.query = typed
			next, cmd := a.findBox(a.searchBack)
			return next.(App).mirrored(), cmd
		case kind == "/" || kind == "?":
			a.diff.query = typed
			next, cmd := a.matchNext(a.searchBack)
			return next.(App).mirrored(), cmd
		}
		return a.command(typed)
	}
	var cmd tea.Cmd
	a.prompt, cmd = a.prompt.Update(msg)
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
	if n, err := strconv.Atoi(typed); err == nil && a.screen == screenDiff && a.review != nil {
		a = a.leave()
		a.focus = mainPanel
		next, cmd := a.toLine(n)
		return next.(App).mirrored(), cmd
	}
	if arg, ok := strings.CutPrefix(typed, "pull "); ok {
		return a.pullCommand(strings.TrimSpace(arg)), nil
	}
	return a.say("not a command: :"+typed, true), nil
}

func (a App) relaid() App {
	if a.screen == screenHome {
		return a.homeRelaid()
	}
	if a.width == 0 || a.review == nil {
		return a
	}
	a = a.sidesBuilt()
	h := a.mainHeight()
	switch {
	case a.screen == screenDiagram:
		a = a.diagramShown()
	case a.screen == screenDiff && a.diff.files != nil:
		a.diff.width = a.mainWidth()
		a.diff = a.diff.rebuilt(*a.review, a.filter, h)
	}
	a.rv = a.rv.rebuilt(*a.review, a.mainWidth(), h, a.p, a.md, a.opts.Location)
	return a.mirrored()
}

func (a App) refresh() (tea.Model, tea.Cmd) {
	if a.screen == screenHome {
		return a.readHome()
	}
	if a.review == nil {
		a.loading, a.failed = true, ""
		return a.say("Opening the review again…", false), loadReview(a.opts.Server, a.opening)
	}
	cmd := reloadReview(a.opts.Server, a.review.id, true)
	if a.liveFor == "" {
		var listening tea.Cmd
		a, listening = a.listenTo(a.review.id)
		cmd = tea.Batch(cmd, listening)
	}
	return a.say("Loading the review again…", false), cmd
}

func (a App) say(notice string, bad bool) App {
	a.notice, a.bad = notice, bad
	return a
}

func (a App) layout() layout {
	var wants [sideCount + 1]int
	for n := reviewPanel; n <= sideCount; n++ {
		wants[n] = max(1, len(a.sides[n].rows))
	}
	return lay(a.width, a.height, a.focus, a.driving, a.mode, wants)
}

func (a App) mainRect() rect {
	if r := a.layout().main; r.shown() {
		return r.inner()
	}
	a.focus = mainPanel
	return a.layout().main.inner()
}

func (a App) mainHeight() int {
	return max(1, a.mainRect().h)
}

func (a App) mainWidth() int {
	return max(1, a.mainRect().w)
}

func (a App) sideHeight(n int) int {
	return max(1, a.layout().sides[n].inner().h)
}

func (a App) menuRows() int {
	return max(1, min(a.height*2/3, 24)-2)
}

func loadReview(c *server.Client, id string) tea.Cmd {
	return func() tea.Msg {
		state, err := c.State(id)
		if err != nil {
			return reviewLoaded{id: id, err: fmt.Errorf("could not open %s: %w", id, err)}
		}
		page, err := c.Page(id)
		if err != nil {
			return reviewLoaded{id: id, err: fmt.Errorf("could not open %s: %w", id, err)}
		}
		return reviewLoaded{id: id, data: reviewData{id: id, state: state, page: page}}
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
