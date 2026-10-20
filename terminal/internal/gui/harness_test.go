package gui

import (
	"bytes"
	stdcontext "context"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/common"
	"github.com/connorb645/lazyreview/terminal/internal/config"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
)

var update = flag.Bool("update", false, "write the screens the tests compare against")

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.Ascii)
	os.Exit(m.Run())
}

func testCommon() *common.Common {
	return commonFor(true)
}

// commonFor is what every layer shares in a test: the default config, as
// the lazyreview command hands it over and terminal/testdata/config.json
// keeps it, drawn for a dark or a light terminal.
func commonFor(dark bool) *common.Common {
	return commonFrom(defaultHandOver(), dark)
}

func defaultHandOver() string {
	text, err := os.ReadFile(filepath.Join("..", "..", "testdata", "config.json"))
	if err != nil {
		panic("could not read the default config: " + err.Error())
	}
	return string(text)
}

func handedOver(t *testing.T, keys map[string]map[string][]string) *common.Common {
	t.Helper()
	var handOver map[string]any
	if err := json.Unmarshal([]byte(defaultHandOver()), &handOver); err != nil {
		t.Fatal(err)
	}
	binding := handOver["keybinding"].(map[string]any)
	for place, actions := range keys {
		for action, sequences := range actions {
			binding[place].(map[string]any)[action] = sequences
		}
	}
	text, err := json.Marshal(handOver)
	if err != nil {
		t.Fatal(err)
	}
	return commonFrom(string(text), true)
}

func commonFrom(handOver string, dark bool) *common.Common {
	tr := i18n.EnglishTranslationSet()
	given, err := config.Decode(handOver, tr)
	if err != nil {
		panic("could not read the config: " + err.Error())
	}
	colours := given.Theme.Light
	if dark {
		colours = given.Theme.Dark
	}
	return &common.Common{Tr: tr, Config: given, Theme: theme.New(colours, dark, lipgloss.ColorProfile())}
}

type answer struct {
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

func recorded(t *testing.T) map[string]answer {
	t.Helper()
	text, err := os.ReadFile(filepath.Join("..", "..", "testdata", "record.json"))
	if err != nil {
		t.Fatalf("could not read the recorded answers: %v", err)
	}
	var answers map[string]answer
	if err := json.Unmarshal(text, &answers); err != nil {
		t.Fatalf("could not read the recorded answers: %v", err)
	}
	return answers
}

type changes struct {
	mu      sync.Mutex
	answers map[string]answer
}

func (c *changes) set(key string, a answer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.answers[key] = a
}

func (c *changes) get(key string) (answer, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	a, ok := c.answers[key]
	return a, ok
}

func served(t *testing.T, changed *changes) *server.Client {
	t.Helper()
	answers := recorded(t)
	stand := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet && r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"changes come only from the app's own pages, or a client with its token"}`))
			return
		}
		key := r.Method + " " + r.URL.EscapedPath()
		if q := r.URL.Query(); len(q) > 0 {
			key += "?" + q.Encode()
		}
		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			key += " " + sortedFields(t, r.Body)
		}
		answer, ok := changed.get(key)
		if !ok {
			answer, ok = answers[key]
		}
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"nothing recorded at ` + strings.ReplaceAll(key, `"`, `'`) + `"}`))
			return
		}
		w.WriteHeader(answer.Status)
		_, _ = w.Write(answer.Body)
	})
	return servedBy(t, stand)
}

// servedBy is a client of a stand-in server that answers with a handler in
// the test's own process.
func servedBy(t *testing.T, handler http.Handler) *server.Client {
	t.Helper()
	file := filepath.Join(t.TempDir(), "server.json")
	record, _ := json.Marshal(server.Record{Address: "http://stand-in", Token: "test-token"})
	if err := os.WriteFile(file, record, 0o600); err != nil {
		t.Fatal(err)
	}
	return server.NewClientThrough(testCommon(), file, inProcess{handler: handler})
}

func sortedFields(t *testing.T, body io.Reader) string {
	var fields map[string]any
	if err := json.NewDecoder(body).Decode(&fields); err != nil {
		t.Errorf("a change sent no JSON: %v", err)
	}
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(fields)
	return strings.TrimSpace(out.String())
}

func opened(t *testing.T, review string, width, height int, set ...func(*Options)) (App, *[]string) {
	t.Helper()
	a, visited, _ := openedWith(t, review, width, height, set...)
	return a, visited
}

func openedWith(t *testing.T, review string, width, height int, set ...func(*Options)) (App, *[]string, *changes) {
	t.Helper()
	changed := &changes{answers: map[string]answer{}}
	a, visited := openedOn(t, changed, review, width, height, set...)
	return a, visited, changed
}

func openedOn(t *testing.T, changed *changes, review string, width, height int, set ...func(*Options)) (App, *[]string) {
	t.Helper()
	return openedIn(t, testCommon(), changed, review, width, height, set...)
}

func openedIn(t *testing.T, cmn *common.Common, changed *changes, review string, width, height int, set ...func(*Options)) (App, *[]string) {
	t.Helper()
	var visited []string
	opts := Options{
		Reviews: served(t, changed), Review: review,
		Open:     func(address string) error { visited = append(visited, address); return nil },
		Now:      func() time.Time { return time.Date(2026, 10, 15, 14, 0, 0, 0, time.UTC) },
		Location: time.UTC,
		Exec:     func(cmd *exec.Cmd, done tea.ExecCallback) tea.Cmd { return func() tea.Msg { return done(cmd.Run()) } },
		Listen:   heardNothing,
		Tick:     func(time.Duration, func(time.Time) tea.Msg) tea.Cmd { return nil },
	}
	for _, s := range set {
		s(&opts)
	}
	a := New(cmn, opts)
	a = drive(t, a, a.Init())
	a = send(t, a, tea.WindowSizeMsg{Width: width, Height: height})
	return a, &visited
}

func send(t *testing.T, a App, msg tea.Msg) App {
	t.Helper()
	next, cmd := a.Update(msg)
	return drive(t, next.(App), cmd)
}

func drive(t *testing.T, a App, cmd tea.Cmd) App {
	t.Helper()
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch msg := c().(type) {
		case nil:
		case tea.BatchMsg:
			queue = append(queue, msg...)
		case tea.QuitMsg:
			return a
		default:
			next, more := a.Update(msg)
			a = next.(App)
			queue = append(queue, more)
		}
	}
	return a
}

func heardNothing(stdcontext.Context, types.Reviews, string) <-chan tea.Msg {
	none := make(chan tea.Msg)
	close(none)
	return none
}

func (a App) with(s context.ContextTree) App {
	a.ContextTree = s
	return a
}

func showBox(a App, at types.DiagramPlace) App {
	return a.with(a.helpers.Diagram.ShowBox(a.ContextTree, at))
}

func relaid(a App) App {
	return a.with(a.helpers.Refresh.Relaid(a.ContextTree))
}

func withEditor(a App, set func(*Options)) App {
	var opts Options
	set(&opts)
	c := *a.c
	c.Editor = opts.Editor
	a.c, a.helpers, a.controllers = &c, helpers.NewHelpers(&c), controllers.NewControllers(&c)
	return a
}

func writer(t *testing.T, text string) (func(*Options), string) {
	t.Helper()
	dir := t.TempDir()
	seen := filepath.Join(dir, "seen.md")
	script := filepath.Join(dir, "editor")
	body := "#!/bin/sh\ncp \"$1\" " + seen + "\n{ printf '%s\\n' '" + text + "'; cat \"$1\"; } > \"$1.new\" && mv \"$1.new\" \"$1\"\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return func(o *Options) { o.Editor = script }, seen
}

func press(t *testing.T, a App, names ...string) App {
	t.Helper()
	for _, n := range names {
		a = send(t, a, keyMsg(n))
	}
	return a
}

func keyMsg(name string) tea.KeyMsg {
	switch name {
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "ctrl+o":
		return tea.KeyMsg{Type: tea.KeyCtrlO}
	case "ctrl+d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	case "ctrl+u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
}

func typed(t *testing.T, a App, text string) App {
	t.Helper()
	for _, r := range text {
		a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return send(t, a, tea.KeyMsg{Type: tea.KeyEnter})
}

func screenIs(t *testing.T, a App, name string) {
	t.Helper()
	got := strings.Join(trimmed(strings.Split(a.View(), "\n")), "\n") + "\n"
	path := filepath.Join("testdata", "screens", name+".txt")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no screen %s yet; write it with go test ./... -update: %v", name, err)
	}
	if string(want) != got {
		t.Errorf("screen %s differs.\n--- kept\n%s\n--- now\n%s", name, want, got)
	}
}

func trimmed(lines []string) []string {
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return lines
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read %s: %v", path, err)
	}
	return string(text)
}
