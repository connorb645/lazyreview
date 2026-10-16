// Package app is the terminal app: the screens and the keys, drawn from what
// the Review Desk server answers. It decides nothing the web app also needs.
package app

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"review-desk/terminal/internal/keys"
	"review-desk/terminal/internal/server"
)

const doubleClick = 400 * time.Millisecond

// App is the terminal app's state: which screen it shows, what the server
// gave it, where the cursor is, and what has been typed toward a key
// sequence. Each key press is Update, which returns the next App.
type App struct {
	opts      Options
	p         palette
	md        *markdown
	width     int
	height    int
	screen    screen
	reviews   reviewsList
	review    *reviewData
	rv        reviewView
	keys      keys.Reader
	panel     *panel
	prompt    textinput.Model
	prompting string
	notice    string
	bad       bool
	loading   bool
	clickedAt time.Time
	clickedY  int
}

// New returns the app, which opens on the review opts names, or on the list
// of reviews.
func New(opts Options) App {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Location == nil {
		opts.Location = time.Local
	}
	if opts.Open == nil {
		opts.Open = openInBrowser
	}
	a := App{opts: opts, p: newPalette(opts.Dark), md: newMarkdown(opts.Dark), loading: true, rv: reviewView{folded: map[string]bool{}}}
	if opts.Review != "" {
		a.screen = screenReview
	}
	return a
}

// Init asks the server for the review to open, or for the list of reviews.
func (a App) Init() tea.Cmd {
	if a.opts.Review != "" {
		return loadReview(a.opts.Server, a.opts.Review)
	}
	return loadReviews(a.opts.Server)
}

// Update takes one message, such as a key, a click, or an answer from the
// server, and returns the app as it then stands.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a.relaid(), nil
	case reviewsLoaded:
		a.loading = false
		if msg.err != nil {
			return a.say(msg.err.Error(), true), nil
		}
		a.reviews.reviews = msg.reviews
		return a.relaid(), nil
	case reviewLoaded:
		a.loading = false
		if msg.err != nil {
			return a.say(msg.err.Error(), true), nil
		}
		a.review = &msg.data
		a.screen = screenReview
		return a.relaid(), nil
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
	if a.panel != nil {
		switch name {
		case "j", "down":
			moved := a.panel.move(1, a.panelRows())
			a.panel = &moved
			return a, nil
		case "k", "up":
			moved := a.panel.move(-1, a.panelRows())
			a.panel = &moved
			return a, nil
		case "q", "esc":
			a.panel = nil
			return a, nil
		}
	}
	var res keys.Result
	a.keys, res = a.keys.Read(name, a.bindings())
	if !res.Waiting {
		a.notice, a.bad = "", false
	}
	if res.Action == "" {
		return a, nil
	}
	return a.do(res.Action, res.Count)
}

func (a App) bindings() keys.Bindings {
	if a.screen == screenReview {
		return everyView.With(reviewKeys)
	}
	return everyView.With(listKeys)
}

func (a App) do(action string, count int) (tea.Model, tea.Cmd) {
	h := a.bodyHeight()
	switch action {
	case "quit":
		return a, tea.Quit
	case "command", "find":
		return a.ask(map[string]string{"command": ":", "find": "/"}[action]), nil
	case "keys":
		return a.showKeys(), nil
	case "close":
		a.panel = nil
		return a, nil
	case "reviews":
		a.screen, a.loading = screenReviews, true
		return a.relaid(), loadReviews(a.opts.Server)
	case "review":
		if a.review == nil {
			return a.say("no review is open; Enter on one opens it", true), nil
		}
		a.screen = screenReview
		return a.relaid(), nil
	case "github":
		return a.openGitHub()
	case "web":
		return a.openWeb()
	}
	if a.screen == screenReviews {
		return a.listAction(action, count, h)
	}
	return a.reviewAction(action, count, h)
}

func (a App) listAction(action string, count, h int) (tea.Model, tea.Cmd) {
	v := a.reviews.view
	switch action {
	case "down":
		a.reviews.view = v.move(count, h)
	case "up":
		a.reviews.view = v.move(-count, h)
	case "top":
		a.reviews.view = v.to(0, h)
	case "bottom":
		a.reviews.view = v.to(len(v.items)-1, h)
	case "open":
		return a.openReview()
	case "next match", "previous match":
		i := a.reviews.match(action == "previous match")
		if i < 0 {
			return a.say(fmt.Sprintf("no review holds %q", a.reviews.query), true), nil
		}
		a.reviews.view = v.to(i, h)
	}
	return a, nil
}

func (a App) reviewAction(action string, count, h int) (tea.Model, tea.Cmd) {
	if a.review == nil {
		return a, nil
	}
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

func (a App) openReview() (tea.Model, tea.Cmd) {
	v := a.reviews.view
	if v.cursor >= len(v.items) {
		return a, nil
	}
	id := v.items[v.cursor].section
	if a.review != nil && a.review.id == id {
		a.screen = screenReview
		return a.relaid(), nil
	}
	a.loading = true
	a.screen = screenReview
	a.review = nil
	a.rv = reviewView{folded: map[string]bool{}}
	return a, loadReview(a.opts.Server, id)
}

func (a App) openGitHub() (tea.Model, tea.Cmd) {
	address := ""
	switch {
	case a.screen == screenReviews:
		if v := a.reviews.view; v.cursor < len(v.items) {
			address = v.items[v.cursor].url
		}
	case a.review != nil:
		if v := a.rv.view; v.cursor < len(v.items) {
			address = v.items[v.cursor].url
		}
		if address == "" && len(a.review.state.Members) > 0 {
			address = a.review.state.Members[0].URL
		}
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
	rows := []helpRow{}
	if a.screen == screenReview {
		rows = append(rows, reviewHelp...)
	} else {
		rows = append(rows, listHelp...)
	}
	rows = append(rows, everyViewHelp...)
	pn := panel{title: "Keys of this view"}
	for _, r := range rows {
		pn.rows = append(pn.rows, panelRow{left: r.keys, right: r.words})
	}
	a.panel = &pn
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
		if kind == "/" {
			a.reviews.query = typed
			return a.listAction("next match", 1, a.bodyHeight())
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
	}
	return a.say("not a command: :"+typed, true), nil
}

func (a App) mouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	h := a.bodyHeight()
	if msg.Action != tea.MouseActionPress {
		return a, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		by := 3
		if msg.Button == tea.MouseButtonWheelUp {
			by = -3
		}
		if a.screen == screenReview {
			a.rv.view = a.rv.view.scroll(by, h)
		} else {
			a.reviews.view = a.reviews.view.scroll(by, h)
		}
		return a, nil
	case tea.MouseButtonLeft:
		y := msg.Y - 1
		if y < 0 || y >= h {
			return a, nil
		}
		now := a.opts.Now()
		double := y == a.clickedY && now.Sub(a.clickedAt) < doubleClick
		a.clickedAt, a.clickedY = now, y
		if a.screen == screenReview {
			if i := a.rv.view.at(y); i >= 0 {
				a.rv.view = a.rv.view.to(i, h)
			}
			return a, nil
		}
		if i := a.reviews.view.at(y); i >= 0 {
			a.reviews.view = a.reviews.view.to(i, h)
			if double {
				return a.openReview()
			}
		}
	}
	return a, nil
}

// relaid draws the screen's lines again for its width, keeping the cursor.
func (a App) relaid() App {
	if a.width == 0 {
		return a
	}
	h := a.bodyHeight()
	if a.screen == screenReview && a.review != nil {
		a.rv = a.rv.rebuilt(*a.review, a.width, h, a.p, a.md, a.opts.Location)
		return a
	}
	lines, items := a.reviews.build(a.width, a.p, a.opts.Location)
	a.reviews.view = a.reviews.view.reset(lines, items, sameItem, h)
	return a
}

func (a App) say(notice string, bad bool) App {
	a.notice, a.bad = notice, bad
	return a
}

func (a App) bodyHeight() int {
	return max(1, a.height-3)
}

func (a App) panelRows() int {
	return max(1, min(a.bodyHeight()*2/3, 20)-3)
}

func loadReviews(c *server.Client) tea.Cmd {
	return func() tea.Msg {
		reviews, err := c.Reviews()
		return reviewsLoaded{reviews: reviews, err: err}
	}
}

func loadReview(c *server.Client, id string) tea.Cmd {
	return func() tea.Msg {
		state, err := c.State(id)
		if err != nil {
			return reviewLoaded{err: fmt.Errorf("could not open %s: %w", id, err)}
		}
		page, err := c.Page(id)
		if err != nil {
			return reviewLoaded{err: fmt.Errorf("could not open %s: %w", id, err)}
		}
		return reviewLoaded{data: reviewData{id: id, state: state, page: page}}
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
