package app

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"review-desk/terminal/internal/server"
)

var update = flag.Bool("update", false, "write the screens the tests compare against")

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.Ascii)
	os.Exit(m.Run())
}

// served starts a stand-in for the server that gives the answers the Node
// tests recorded from a real one, and returns a client of it. Like the
// server, it takes a change only with the token.
func served(t *testing.T) *server.Client {
	t.Helper()
	text, err := os.ReadFile(filepath.Join("..", "..", "testdata", "record.json"))
	if err != nil {
		t.Fatalf("could not read the recorded answers: %v", err)
	}
	var answers map[string]struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(text, &answers); err != nil {
		t.Fatalf("could not read the recorded answers: %v", err)
	}
	stand := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		answer, ok := answers[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"nothing recorded at ` + strings.ReplaceAll(key, `"`, `'`) + `"}`))
			return
		}
		w.WriteHeader(answer.Status)
		_, _ = w.Write(answer.Body)
	}))
	t.Cleanup(stand.Close)
	file := filepath.Join(t.TempDir(), "server.json")
	record, _ := json.Marshal(server.Record{Address: stand.URL, Token: "test-token"})
	if err := os.WriteFile(file, record, 0o600); err != nil {
		t.Fatal(err)
	}
	return server.NewClient(file)
}

// sortedFields is a change's JSON fields with their keys sorted, as the record keys them.
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

// opened is the app on the stand-in server, sized, with everything it asked for at the start answered.
func opened(t *testing.T, review string, width, height int, set ...func(*Options)) (App, *[]string) {
	t.Helper()
	var visited []string
	opts := Options{
		Server: served(t), Review: review, Dark: true,
		Open:     func(address string) error { visited = append(visited, address); return nil },
		Now:      func() time.Time { return time.Date(2026, 10, 15, 14, 0, 0, 0, time.UTC) },
		Location: time.UTC,
		Exec:     func(cmd *exec.Cmd, done tea.ExecCallback) tea.Cmd { return func() tea.Msg { return done(cmd.Run()) } },
	}
	for _, s := range set {
		s(&opts)
	}
	a := New(opts)
	a = drive(t, a, a.Init())
	a = send(t, a, tea.WindowSizeMsg{Width: width, Height: height})
	return a, &visited
}

// send gives the app a message, then every message its commands give back, until none is left.
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

// writer is an editor that writes text above the file's cut line, and
// keeps a copy of the file as it was given, at the path it returns.
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

// press types keys, each named as the app names keys, such as "j", "space" or "enter".
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

// typed types text into the command line or a search, then Enter.
func typed(t *testing.T, a App, text string) App {
	t.Helper()
	for _, r := range text {
		a = send(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return send(t, a, tea.KeyMsg{Type: tea.KeyEnter})
}

// screenIs compares the whole screen with the one kept under testdata/screens.
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
