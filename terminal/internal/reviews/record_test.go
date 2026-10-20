package reviews

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/ask"
	"github.com/connorb645/lazyreview/terminal/internal/commands/claude"
	"github.com/connorb645/lazyreview/terminal/internal/commands/git"
	"github.com/connorb645/lazyreview/terminal/internal/commands/github"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/queue"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/testsupport"
)

type recorded struct {
	key    string
	status int
	body   json.RawMessage
}

// recordedAnswers is every answer of the record, in the order it was
// recorded, as the changes among them were made in turn.
func recordedAnswers(t *testing.T) []recorded {
	t.Helper()
	text, err := os.ReadFile(testsupport.Testdata("record.json"))
	if err != nil {
		t.Fatalf("could not read the record: %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(text))
	if _, err := decoder.Token(); err != nil {
		t.Fatal(err)
	}
	var out []recorded
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			t.Fatal(err)
		}
		var answer struct {
			Status int             `json:"status"`
			Body   json.RawMessage `json:"body"`
		}
		if err := decoder.Decode(&answer); err != nil {
			t.Fatal(err)
		}
		out = append(out, recorded{key: key.(string), status: answer.Status, body: answer.Body})
	}
	return out
}

// standInHome is the stand-in data home's place as the record names it.
const standInHome = "/home/reviewer/.lazyreview"

func TestMain(m *testing.M) {
	testsupport.ActAsFakeClaude()
	os.Exit(m.Run())
}

// madeCopies stands in for the copies of the code, which the stand-in data
// home has no clone to make from: each is an empty folder.
type madeCopies struct{}

func (madeCopies) Prepare(_ context.Context, folder string, _ git.CopyOf) (string, error) {
	return folder, os.MkdirAll(folder, 0o755)
}

func (madeCopies) Prune(string, time.Duration, string) {}

func standInReviews(t *testing.T) (*Reviews, *testsupport.FakeClaude, string) {
	tr := i18n.EnglishTranslationSet()
	gh := testsupport.FakeGhFrom(t, testsupport.Testdata("github.json"))
	root := testsupport.StandInHome(t)
	locks := store.NewLocks(tr)
	reviews := store.NewReviewStore(tr, store.DataHome{Root: root}, locks)
	now := func() string { return "2026-10-15T13:00:00.000Z" }
	q := queue.NewService(tr, reviews, func(host, account string) *github.GitHub { return github.New(tr, host, account, gh.Run) }, now)
	fake := testsupport.NewFakeClaude(t)
	fake.Knows("00000000-0000-4000-8000-000000000001")
	asking := ask.NewService(tr, reviews, locks, claude.New(tr, fake.Command, ""), madeCopies{}, nil, nil, now)
	return New(tr, reviews, q, asking, nil, nil, nil), fake, root
}

// Each answer the terminal app is tested against, of what the Go code now
// gives in the server's place, is the one the server recorded: read from
// the same data home, with each change of the record made in turn. Only a
// state's version, a short hash of how the state is written, and when a
// review was sent may differ.
func TestTheGoCodeGivesTheAnswersTheServerRecorded(t *testing.T) {
	r, fake, root := standInReviews(t)
	checked := 0
	for _, answer := range recordedAnswers(t) {
		if strings.HasPrefix(answer.key, "POST /api/reviews/github.com/octo/widgets/21/threads/9/stop ") {
			leftWaiting(t, r)
		}
		got, ok := r.answer(t, answer.key)
		if !ok {
			continue
		}
		checked++
		if answer.status != 200 {
			var refused struct {
				Error string `json:"error"`
			}
			_ = json.Unmarshal(answer.body, &refused)
			if got.err == nil || got.err.Error() != refused.Error {
				t.Errorf("%s\n  recorded the refusal %q\n  Go gives %v", answer.key, refused.Error, got.err)
			}
			continue
		}
		if got.err != nil {
			t.Errorf("%s\n  recorded an answer\n  Go refuses: %v", answer.key, got.err)
			continue
		}
		want := reflect.New(reflect.TypeOf(got.value).Elem()).Interface()
		if err := json.Unmarshal(answer.body, want); err != nil {
			t.Fatalf("%s: could not read the recorded answer: %v", answer.key, err)
		}
		if a, b := comparable(t, want), comparable(t, got.value); !reflect.DeepEqual(a, b) {
			t.Errorf("%s answers otherwise:\n%s", answer.key, firstDifference("", a, b))
		}
	}
	if checked != 29 {
		t.Fatalf("%d of the record's answers are the Go code's, not 29", checked)
	}
	var recorded []ask.Question
	text, err := os.ReadFile(testsupport.Testdata("asked.json"))
	if err == nil {
		err = json.Unmarshal(text, &recorded)
	}
	if err != nil || len(recorded) != 3 {
		t.Fatalf("could not read the questions Claude was asked: %v", err)
	}
	ref, _ := models.ParseReviewRef(r.tr, "github.com/octo/widgets/21")
	rev, _ := r.store.Read(ref)
	read, _ := r.store.ReadPage(ref)
	threads, _ := r.store.Threads(ref)
	first := threads[0]
	first.Messages = first.Messages[:1]
	asked := []ask.Question{ask.QuestionFor(r.tr, *rev, *read.Page, first, r.store.Home().Copy(ref, 1))}
	for _, call := range fake.Asked() {
		asked = append(asked, ask.Question{System: call.Args[slices.Index(call.Args, "--append-system-prompt")+1], Prompt: call.Prompt})
	}
	for i := range asked {
		asked[i].Prompt = strings.ReplaceAll(asked[i].Prompt, root, standInHome)
	}
	if !reflect.DeepEqual(asked, recorded) {
		for i := range recorded {
			if i < len(asked) && asked[i] != recorded[i] {
				t.Errorf("question %d was asked as:\n%s\nnot as recorded:\n%s", i+1, asked[i].Prompt, recorded[i].Prompt)
			}
		}
		t.Fatalf("Go asked %d questions, the record holds %d", len(asked), len(recorded))
	}
}

// leftWaiting is the conversation the recorder leaves waiting, as after the
// server stopped mid-answer, before it stops it.
func leftWaiting(t *testing.T, r *Reviews) {
	t.Helper()
	ref, _ := models.ParseReviewRef(r.tr, "github.com/octo/widgets/21")
	left := models.Conversation{ID: 9, Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 3, CreatedAt: "2026-10-15T13:00:08.000Z", Session: "00000000-0000-4000-8000-000000000003",
		Messages: []models.Message{{From: "you", Text: "Does anything test this?", At: "2026-10-15T13:00:09.000Z"}},
		Working:  &models.Working{Steps: []models.Step{{Words: "Read Sources/Basket/Basket.swift"}, {Words: "Ran swift test"}}}}
	write, err := r.store.ThreadWrite(ref, left)
	if err == nil {
		err = os.WriteFile(write.Path, []byte(write.Text), 0o644)
	}
	if err != nil {
		t.Fatal(err)
	}
}

// Each conversation of the stand-in data home, written back by the Go code,
// is the file the server wrote, to the byte.
func TestAConversationIsWrittenBackAsTheServerWroteIt(t *testing.T) {
	r, _, _ := standInReviews(t)
	ref, _ := models.ParseReviewRef(r.tr, "github.com/octo/widgets/21")
	threads, err := r.store.Threads(ref)
	if err != nil || len(threads) == 0 {
		t.Fatalf("the stand-in conversations read %v, %v", threads, err)
	}
	for _, c := range threads {
		write, err := r.store.ThreadWrite(ref, c)
		was, _ := os.ReadFile(write.Path)
		if err != nil || write.Text != string(was) {
			t.Errorf("conversation %d is written as:\n%s\nnot as:\n%s", c.ID, write.Text, was)
		}
	}
}

type given struct {
	value any
	err   error
}

type stateAnswer struct {
	State models.State `json:"state"`
}

type threadAnswer struct {
	Thread models.Conversation `json:"thread"`
}

type postAnswer struct {
	Message string       `json:"message"`
	State   models.State `json:"state"`
}

// answer is what the Go code gives for one of the record's keys, as the
// terminal app's client reads it, or false for a key the server still
// answers.
func (r *Reviews) answer(t *testing.T, key string) (given, bool) {
	t.Helper()
	method, rest, _ := strings.Cut(key, " ")
	address, body, _ := strings.Cut(rest, " {")
	if body != "" {
		body = "{" + body
	}
	path, rawQuery, _ := strings.Cut(address, "?")
	query, _ := url.ParseQuery(rawQuery)
	var fields map[string]any
	if body != "" {
		if err := json.Unmarshal([]byte(body), &fields); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
	}
	reviewPath, ok := strings.CutPrefix(path, "/api/reviews/")
	if !ok {
		return given{}, false
	}
	parts := strings.Split(reviewPath, "/")
	var id, tail string
	switch {
	case parts[0] == "local" && len(parts) >= 3:
		id, tail = strings.Join(parts[:3], "/"), strings.Join(parts[3:], "/")
	case len(parts) >= 4:
		id, tail = strings.Join(parts[:4], "/"), strings.Join(parts[4:], "/")
	default:
		return given{}, false
	}
	id, _ = url.PathUnescape(id)
	text := func(name string) string {
		if v, ok := fields[name].(string); ok {
			return v
		}
		return query.Get(name)
	}
	number := func(name string) int {
		if v, ok := fields[name].(float64); ok {
			return int(v)
		}
		n, _ := strconv.Atoi(query.Get(name))
		return n
	}
	lines := models.Lines{Path: text("path"), Side: text("side"), Line: number("line"), StartLine: number("start_line"), StartSide: text("start_side")}
	if lines.StartLine != 0 && lines.StartSide == "" {
		lines.StartSide = lines.Side
	}
	comment, isComment := strings.CutPrefix(tail, "comments/")
	switch {
	case method == "GET" && tail == "":
		v, err := r.State(id)
		return given{&v, err}, true
	case method == "GET" && tail == "page":
		v, err := r.Page(id)
		return given{&v, err}, true
	case method == "GET" && tail == "file":
		v, err := r.File(id, text("path"), text("build"))
		return given{&v, err}, true
	case method == "GET" && tail == "where":
		v, err := r.Where(id, lines, number("pull"), text("build"))
		return given{&v, err}, true
	case method == "POST" && tail == "comments":
		v, err := r.Queue(id, lines, number("pull"), text("body"), text("build"))
		return given{&stateAnswer{v}, err}, true
	case method == "PATCH" && isComment:
		n, _ := strconv.Atoi(comment)
		v, err := r.Edit(id, n, text("body"), text("version"), text("build"))
		return given{&stateAnswer{v}, err}, true
	case method == "DELETE" && isComment:
		n, _ := strconv.Atoi(comment)
		v, err := r.Drop(id, n, text("version"), text("build"))
		return given{&stateAnswer{v}, err}, true
	case method == "POST" && tail == "threads":
		v, err := r.Ask(id, lines, text("text"), text("build"))
		return given{&threadAnswer{v}, err}, true
	case method == "POST" && strings.HasPrefix(tail, "threads/") && strings.HasSuffix(tail, "/messages"):
		n, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(tail, "threads/"), "/messages"))
		v, err := r.Reply(id, n, text("text"), text("build"))
		return given{&threadAnswer{v}, err}, true
	case method == "POST" && strings.HasPrefix(tail, "threads/") && strings.HasSuffix(tail, "/stop"):
		n, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(tail, "threads/"), "/stop"))
		v, err := r.Stop(id, n)
		return given{&threadAnswer{v}, err}, true
	case method == "POST" && tail == "post":
		message, v, err := r.Post(id, number("pull"), text("action"), text("set"), text("build"))
		return given{&postAnswer{message, v}, err}, true
	}
	return given{}, false
}

// comparable is an answer as plain values, with its state's version, the
// times reviews were sent and a conversation's times and session left out,
// and an empty list the same as none.
func comparable(t *testing.T, v any) any {
	t.Helper()
	text, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var plain any
	if err := json.Unmarshal(text, &plain); err != nil {
		t.Fatal(err)
	}
	if m, ok := plain.(map[string]any); ok && m["thread"] != nil {
		return maskedConversation(plain)
	}
	return masked(plain, "")
}

func maskedConversation(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, inner := range x {
			if k != "at" && k != "created_at" && k != "session" {
				out[k] = maskedConversation(inner)
			}
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, inner := range x {
			out[i] = maskedConversation(inner)
		}
		return out
	}
	return v
}

func masked(v any, key string) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, inner := range x {
			if k == "version" && key == "state" || k == "at" && key == "posted" {
				continue
			}
			out[k] = masked(inner, k)
		}
		if key == "" {
			if _, ok := out["queue"]; ok {
				delete(out, "version")
			}
		}
		return out
	case []any:
		if len(x) == 0 {
			return nil
		}
		out := make([]any, len(x))
		for i, inner := range x {
			out[i] = masked(inner, key)
		}
		return out
	}
	return v
}

func firstDifference(at string, a, b any) string {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok {
			break
		}
		for k := range x {
			if !reflect.DeepEqual(x[k], y[k]) {
				return firstDifference(at+"."+k, x[k], y[k])
			}
		}
		for k := range y {
			if _, ok := x[k]; !ok {
				return fmt.Sprintf("  %s.%s: recorded nothing, Go gives %v", at, k, y[k])
			}
		}
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			break
		}
		for i := range x {
			if !reflect.DeepEqual(x[i], y[i]) {
				return firstDifference(at+"["+strconv.Itoa(i)+"]", x[i], y[i])
			}
		}
	}
	return fmt.Sprintf("  %s:\n    recorded %v\n    Go gives %v", at, a, b)
}
