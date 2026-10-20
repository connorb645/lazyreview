package reviews

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/ask"
	"github.com/connorb645/lazyreview/terminal/internal/commands/claude"
	"github.com/connorb645/lazyreview/terminal/internal/commands/git"
	"github.com/connorb645/lazyreview/terminal/internal/commands/github"
	"github.com/connorb645/lazyreview/terminal/internal/config"
	"github.com/connorb645/lazyreview/terminal/internal/home"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/loader"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/queue"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/testsupport"
)

const (
	standInHome   = "/home/reviewer/.lazyreview"
	standInOrigin = "/stand-in/origin.git"
	standInConfig = "/home/reviewer/Library/Application Support/lazyreview/config.yml"
	standInUser   = "/home/reviewer"
	stackID       = "github.com/octo/widgets/21"
	singleID      = "github.com/octo/widgets/30"
)

func TestMain(m *testing.M) {
	testsupport.ActAsFakeClaude()
	os.Exit(m.Run())
}

type recorded struct {
	key    string
	status int
	body   json.RawMessage
}

type askedQuestion struct {
	System string `json:"system"`
	Prompt string `json:"prompt"`
}

type recording struct {
	answers []recorded
	home    map[string]string
	routes  string
	asked   []askedQuestion
}

type recordedHomes struct {
	Lists []struct {
		Key    string                     `json:"key"`
		Config *string                    `json:"config"`
		Routes map[string]json.RawMessage `json:"routes"`
	} `json:"lists"`
	Changes []struct {
		Key    string  `json:"key"`
		Before *string `json:"before"`
	} `json:"changes"`
}

type loadAnswer struct {
	ID    string   `json:"id"`
	Notes []string `json:"notes"`
}

type recordStep struct {
	key string
	do  func()
}

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
	asking := ask.NewService(tr, reviews, locks, claude.New(tr, fake.Command, ""), madeCopies{}, nil, nil, nil, now)
	return New(tr, reviews, q, asking, nil, nil, nil, nil), fake, root
}

// The answers the terminal app's tests read, the stand-in data home they
// come from, the fake GitHub's routes and the questions the stand-in Claude
// was asked are the ones the Go code gives now: from its own loads of the
// workbench, with each change made in turn. LAZYREVIEW_RECORD=1 writes them
// again.
func TestTheRecordedAnswersAreTheOnesTheGoCodeGivesNow(t *testing.T) {
	now := record(t).files(t)
	kept := testsupport.Testdata("home")
	if os.Getenv("LAZYREVIEW_RECORD") != "" {
		for name, text := range now {
			path := testsupport.Testdata(name)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		for _, name := range filesUnder(t, kept, "home") {
			if _, made := now[name]; !made {
				if err := os.Remove(testsupport.Testdata(name)); err != nil {
					t.Fatal(err)
				}
			}
		}
		return
	}
	again := "write it again with LAZYREVIEW_RECORD=1 go test ./internal/reviews, then run every Go test"
	names := make([]string, 0, len(now))
	for name := range now {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		was, err := os.ReadFile(testsupport.Testdata(name))
		if err != nil || string(was) != now[name] {
			t.Errorf("testdata/%s is not what the Go code gives now; %s%s", name, again, whereItDiffers(string(was), now[name]))
		}
	}
	for _, name := range filesUnder(t, kept, "home") {
		if _, made := now[name]; !made {
			t.Errorf("testdata/%s is no longer made; %s", name, again)
		}
	}
}

// Each conversation of the stand-in data home, written back by the Go code,
// is the file kept there, to the byte.
func TestAConversationIsWrittenBackAsItWasKept(t *testing.T) {
	r, _, _ := standInReviews(t)
	ref, _ := models.ParseReviewRef(r.tr, stackID)
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

func record(t *testing.T) recording {
	t.Helper()
	tr := i18n.EnglishTranslationSet()
	repo := testsupport.NewWorkbench(t)
	repo.Checkout("fix/wording")
	repo.Commit("fix/wording", map[string]string{
		"docs/guide.md":  "# Guide\n\nThe catalog lists every item.\n\n- Browse\n- Search\n- Keep a basket\n\n## Steps\n\n1. Open the catalog.\n1. Pick an item.\n1. Add it to the basket.\n\n```swift\nlet basket = Basket(catalog: catalog)\n```\n",
		"docs/basket.md": "# Basket\n\nA basket keeps its prices until checkout.\n\n1. Add an item\n2. Remove an item\n",
	}, "More guide")
	repo.Checkout("main")
	origin := repo.Publish(map[int]string{21: "feature/search", 22: "feature/basket", 30: "fix/wording"})
	threads := map[int][]any{
		21: {githubThread("T1", "Sources/Catalog/MemoryCatalog.swift", 9, "sam", "Should this keep the order items were added in?", nil)},
		22: {githubThread("T2", "Sources/Basket/Basket.swift", 7, "sam", "Could the search be passed in?", map[string]any{"isResolved": true, "resolvedBy": map[string]any{"login": "sam"}})},
	}
	routes := testsupport.GitHubRoutes(repo, origin, testsupport.WorkbenchPulls, threads)
	gh := &testsupport.FakeGh{}
	answerAs := func(routes testsupport.Routes) {
		raw, err := routes.Raw()
		if err != nil {
			t.Fatal(err)
		}
		gh.Routes = raw
	}
	answerAs(routes)

	root := t.TempDir()
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	standIn := strings.NewReplacer(resolved, standInHome, root, standInHome)
	locks := store.NewLocks(tr)
	reviews := store.NewReviewStore(tr, store.DataHome{Root: root}, locks)
	ghFor := func(host, account string) *github.GitHub { return github.New(tr, host, account, gh.Run) }
	clock := func(hour int) func() string {
		tick := 0
		return func() string {
			at := time.Date(2026, 10, 15, hour, 0, tick, 0, time.UTC)
			tick++
			return at.Format("2006-01-02T15:04:05.000Z")
		}
	}
	now, askNow := clock(12), clock(13)
	sessions := 0
	session := func() string {
		sessions++
		return fmt.Sprintf("00000000-0000-4000-8000-%012d", sessions)
	}
	fake := testsupport.NewFakeClaude(t)
	asking := ask.NewService(tr, reviews, locks, claude.New(tr, fake.Command, ""), nil, nil, nil, session, askNow)
	loads := loader.NewLoader(tr, reviews, locks, git.NewClones(tr, nil), ghFor, now)
	r := New(tr, reviews, queue.NewService(tr, reviews, ghFor, now), asking, nil, nil, loads, loader.NewBranchLoader(tr, reviews, now))
	link := func(n int) string { return "https://github.com/octo/widgets/pull/" + strconv.Itoa(n) }
	withContext := true
	if _, err := loads.Load(context.Background(), loader.PullsRequest{Links: []string{link(21), link(22)}, Context: &withContext}); err != nil {
		t.Fatal(err)
	}

	var rec recording
	answered := func(key string, value any, err error) {
		t.Helper()
		answer := recorded{key: key, status: 200}
		if err != nil {
			answer.status, value = 422, map[string]string{"error": err.Error()}
		}
		body, err := plainJSON(value)
		if err != nil {
			t.Fatal(err)
		}
		answer.body = json.RawMessage(standIn.Replace(body))
		rec.answers = append(rec.answers, answer)
	}

	var homes recordedHomes
	text, err := os.ReadFile(testsupport.Testdata("homes.json"))
	if err == nil {
		err = json.Unmarshal(text, &homes)
	}
	if err != nil {
		t.Fatalf("could not read what the home screen is listed from: %v", err)
	}
	for _, list := range homes.Lists {
		settings := ""
		if list.Config != nil {
			settings = *list.Config
		}
		read := func() (config.Read, error) { return config.ConfigFrom(tr, settings, standInConfig, standInUser) }
		listed := &testsupport.FakeGh{Routes: list.Routes}
		pulls, err := home.NewOpenPulls(tr, reviews, read, func(host, account string) *github.GitHub { return github.New(tr, host, account, listed.Run) }).List()
		answered(list.Key, pulls, err)
	}
	for _, change := range homes.Changes {
		path := filepath.Join(t.TempDir(), "config.yml")
		if change.Before != nil {
			if err := os.WriteFile(path, []byte(*change.Before), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		read := func() (config.Read, error) {
			text, _ := os.ReadFile(path)
			return config.ConfigFrom(tr, string(text), path, standInUser)
		}
		var body struct {
			Hide  string   `json:"hide"`
			Show  []string `json:"show"`
			Repos []string `json:"repos"`
		}
		_, fields, _ := strings.Cut(change.Key, " {")
		if err := json.Unmarshal([]byte("{"+fields), &body); err != nil {
			t.Fatal(err)
		}
		hidden, err := home.NewHiddenRepositories(tr, read).Change(body.Hide, body.Show, body.Repos)
		if err != nil {
			err = models.Refused(strings.Replace(err.Error(), path, standInConfig, 1))
		}
		hidden.Config = strings.Replace(hidden.Config, path, standInConfig, 1)
		answered(change.Key, hidden, err)
	}

	if _, err := loads.Load(context.Background(), loader.PullsRequest{Links: []string{link(30)}}); err != nil {
		t.Fatal(err)
	}
	stack, single := mustRef(t, tr, stackID), mustRef(t, tr, singleID)
	build := pageBuild(t, reviews, stack)
	must := func(_ any, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(r.Queue(stackID, models.Lines{Path: "Sources/Catalog/MemoryCatalog.swift", Side: "RIGHT", Line: 10, StartLine: 8, StartSide: "RIGHT"}, 0, "Keep the order items came in.", build))
	must(r.Queue(stackID, models.Lines{Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 11, StartLine: 9, StartSide: "LEFT"}, 0, "Why one?", build))
	fake.Answer(testsupport.FakeAnswer{Text: "It could: the basket only reads from it.", Steps: []testsupport.FakeStep{
		{Tool: "Grep", Input: map[string]any{"pattern": "Search("}},
		{Tool: "Bash", Input: map[string]any{"command": `gh pr comment 22 --body "Pass the search in."`}, Refused: "lazyreview refused this: gh may only read, one command at a time."},
	}})
	must(r.Ask(stackID, models.Lines{Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 7}, "Could this be passed in?", build))
	fake.Answer(testsupport.FakeAnswer{Text: "It returns early when the list is empty."})
	must(r.Queue(singleID, models.Lines{Path: "docs/guide.md", Side: "RIGHT", Line: 3}, 0, "Every item, or every item shown?", pageBuild(t, reviews, single)))
	written := "---\nstack: 21, 22\nhead: " + repo.Git("rev-parse", "feature/basket") + "\ntitle: Search, then a basket\n---\n## What this changes\n\n" +
		"#21 adds a search over the catalog in `Search.swift:4`. #22 gives the basket a search of its own, in `Basket.swift:7`. [Likely]\n\n" +
		"## Risks\n\n`MemoryCatalog.swift:9` appends without checking for duplicates. [Guessing]\n"
	if err := reviews.Change([]models.ReviewRef{stack}, func() ([]store.FileWrite, error) {
		return []store.FileWrite{{Path: reviews.Home().Review(stack).Written(), Text: written}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	rec.home = map[string]string{}
	for _, name := range filesUnder(t, filepath.Join(root, "reviews"), "home/reviews") {
		text, err := os.ReadFile(filepath.Join(root, strings.TrimPrefix(name, "home/")))
		if err != nil {
			t.Fatal(err)
		}
		rec.home[name] = string(text)
	}

	give := func(key string) {
		t.Helper()
		got, ok := r.answer(t, key)
		if !ok {
			t.Fatalf("no answer is made for %s", key)
		}
		answered(key, got.value, got.err)
	}
	for _, id := range []string{stackID, singleID} {
		base := "/api/reviews/" + id
		page, err := r.Page(id)
		if err != nil {
			t.Fatal(err)
		}
		give(answerKey("GET", base, nil, nil))
		give(answerKey("GET", base+"/page", nil, nil))
		for _, f := range append(append([]models.FileEntry{}, page.Model.Files...), page.Model.ContextFiles...) {
			give(answerKey("GET", base+"/file", url.Values{"path": {f.Path}, "build": {page.Build}}, nil))
		}
	}
	base, one := "/api/reviews/"+stackID, "/api/reviews/"+singleID
	for _, given := range []url.Values{
		{"path": {"Sources/Basket/Basket.swift"}, "side": {"RIGHT"}, "line": {"3"}},
		{"path": {"Sources/Catalog/MemoryCatalog.swift"}, "side": {"RIGHT"}, "start_line": {"7"}, "start_side": {"RIGHT"}, "line": {"9"}},
		{"path": {"Sources/Catalog/Catalog.swift"}, "side": {"RIGHT"}, "line": {"1"}},
		{"path": {"Sources/Basket/Basket.swift"}, "side": {"RIGHT"}, "line": {"7"}},
	} {
		given.Set("build", build)
		give(answerKey("GET", base+"/where", given, nil))
	}

	state, err := r.State(stackID)
	if err != nil {
		t.Fatal(err)
	}
	version := func(id int) string {
		for _, m := range state.Queue.Members {
			for _, c := range m.Comments {
				if c.ID == id {
					return c.Version
				}
			}
		}
		t.Fatalf("no queued comment %d", id)
		return ""
	}
	leftWaiting := func() {
		created, id := askNow(), session()
		left := models.Conversation{ID: 9, Path: "Sources/Basket/Basket.swift", Side: "RIGHT", Line: 3, CreatedAt: created, Session: id,
			Messages: []models.Message{{From: "you", Text: "Does anything test this?", At: askNow()}},
			Working:  &models.Working{Steps: []models.Step{{Words: "Read Sources/Basket/Basket.swift"}, {Words: "Ran swift test"}}}}
		if err := reviews.Change([]models.ReviewRef{stack}, func() ([]store.FileWrite, error) {
			write, err := reviews.ThreadWrite(stack, left)
			return []store.FileWrite{write}, err
		}); err != nil {
			t.Fatal(err)
		}
	}
	changes := []recordStep{
		{key: answerKey("POST", base+"/comments", nil, map[string]any{"path": "Sources/Basket/Basket.swift", "side": "RIGHT", "line": 3, "body": "Name the search for what it finds.", "build": build})},
		{key: answerKey("POST", base+"/comments", nil, map[string]any{"path": "Sources/Catalog/MemoryCatalog.swift", "side": "RIGHT", "start_line": 7, "start_side": "RIGHT", "line": 9, "body": "One add is enough.", "build": build})},
		{key: answerKey("POST", base+"/comments", nil, map[string]any{"path": "Sources/Basket/Basket.swift", "side": "RIGHT", "line": 7, "body": "It could: the basket only reads from it.", "build": build})},
		{key: answerKey("PATCH", base+"/comments/1", nil, map[string]any{"body": "Keep the order the items came in.", "version": "stale1", "build": build})},
		{key: answerKey("PATCH", base+"/comments/1", nil, map[string]any{"body": "Keep the order the items came in.", "version": version(1), "build": build})},
		{key: answerKey("DELETE", base+"/comments/2", url.Values{"build": {build}, "version": {version(2)}}, nil)},
		{key: answerKey("POST", base+"/threads", nil, map[string]any{"path": "Sources/Basket/Basket.swift", "side": "RIGHT", "line": 3, "text": "Why keep a search here?", "build": build})},
		{key: answerKey("POST", base+"/threads/1/messages", nil, map[string]any{"text": "Could it be passed in?", "build": build})},
		{do: leftWaiting},
		{key: answerKey("POST", base+"/threads/9/stop", nil, map[string]any{})},
	}
	oneState, err := r.State(singleID)
	if err != nil {
		t.Fatal(err)
	}
	oneBuild := pageBuild(t, reviews, single)
	routes.Set("POST repos/octo/widgets/pulls/30/reviews", map[string]any{"id": 8, "state": "APPROVED", "html_url": "https://github.com/octo/widgets/pull/30#pullrequestreview-8"})
	routes.Set("GET repos/octo/widgets/pulls/30/comments?per_page=100&page=1", []any{map[string]any{"pull_request_review_id": 8}})
	answerAs(routes)
	routesText, err := plainJSON(routes)
	if err != nil {
		t.Fatal(err)
	}
	rec.routes = strings.ReplaceAll(routesText, origin, standInOrigin)
	changes = append(changes,
		recordStep{key: answerKey("POST", one+"/post", nil, map[string]any{"pull": 30, "action": "comment", "set": "stale0000000", "build": oneBuild})},
		recordStep{key: answerKey("POST", one+"/post", nil, map[string]any{"pull": 30, "action": "approve-with-comments", "set": oneState.Queue.Members[0].Sets["approve-with-comments"].Set, "build": oneBuild})},
		recordStep{key: answerKey("POST", one+"/comments", nil, map[string]any{"path": "docs/guide.md", "side": "RIGHT", "start_line": 3, "start_side": "LEFT", "line": 3, "body": "Why every item?", "build": oneBuild})},
	)
	for _, given := range []url.Values{
		{"path": {"docs/guide.md"}, "side": {"RIGHT"}, "line": {"3"}},
		{"path": {"docs/guide.md"}, "side": {"RIGHT"}, "start_line": {"3"}, "start_side": {"LEFT"}, "line": {"3"}},
	} {
		given.Set("build", oneBuild)
		give(answerKey("GET", one+"/where", given, nil))
	}
	changes = append(changes,
		recordStep{key: answerKey("POST", one+"/refresh", nil, map[string]any{})},
		recordStep{key: answerKey("POST", "/api/load", nil, map[string]any{"links": []string{link(30)}, "account": "reviewer"})},
		recordStep{key: answerKey("POST", "/api/load", nil, map[string]any{"links": []string{"https://github.com/octo/gadgets/pull/7"}, "account": "reviewer"})},
	)
	for _, change := range changes {
		if change.do != nil {
			change.do()
			continue
		}
		give(change.key)
	}

	for _, call := range fake.Asked() {
		rec.asked = append(rec.asked, askedQuestion{System: call.Args[slices.Index(call.Args, "--append-system-prompt")+1], Prompt: standIn.Replace(call.Prompt)})
	}
	return rec
}

func (rec recording) files(t *testing.T) map[string]string {
	t.Helper()
	var answers bytes.Buffer
	answers.WriteByte('{')
	for i, a := range rec.answers {
		if i > 0 {
			answers.WriteByte(',')
		}
		key, err := plainJSON(a.key)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&answers, `%s:{"status":%d,"body":%s}`, key, a.status, a.body)
	}
	answers.WriteByte('}')
	asked, err := plainJSON(rec.asked)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{"record.json": spaced(t, answers.String()), "github.json": spaced(t, rec.routes), "asked.json": spaced(t, asked)}
	for name, text := range rec.home {
		out[name] = text
	}
	return out
}

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
	if method == "POST" && path == "/api/load" {
		var links []string
		for _, l := range fields["links"].([]any) {
			links = append(links, l.(string))
		}
		id, notes, err := r.Load(links, text("account"))
		return given{&loadAnswer{ID: id, Notes: noneEmpty(notes)}, err}, true
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
	case method == "POST" && tail == "refresh":
		notes, err := r.Refresh(id)
		return given{&loadAnswer{ID: id, Notes: noneEmpty(notes)}, err}, true
	}
	return given{}, false
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

func answerKey(method, path string, query url.Values, fields map[string]any) string {
	key := method + " " + path
	if len(query) > 0 {
		key += "?" + query.Encode()
	}
	if fields != nil {
		text, _ := plainJSON(fields)
		key += " " + text
	}
	return key
}

func githubThread(id, path string, line int, login, body string, extra map[string]any) map[string]any {
	number, _ := strconv.Atoi(id[1:])
	node := map[string]any{
		"id": id, "isResolved": false, "isOutdated": false, "path": path, "line": line, "startLine": nil, "originalLine": line,
		"originalStartLine": nil, "diffSide": "RIGHT", "startDiffSide": nil, "subjectType": "LINE", "resolvedBy": nil,
		"comments": map[string]any{"totalCount": 1, "nodes": []any{map[string]any{
			"databaseId": number, "author": map[string]any{"login": login, "__typename": "User"}, "body": body,
			"createdAt": "2026-10-15T10:00:00Z", "url": "https://github.com/octo/widgets/pull/21#discussion_r" + id[1:], "diffHunk": "",
		}}},
	}
	for k, v := range extra {
		node[k] = v
	}
	return node
}

func mustRef(t *testing.T, tr *i18n.TranslationSet, id string) models.ReviewRef {
	t.Helper()
	ref, err := models.ParseReviewRef(tr, id)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func pageBuild(t *testing.T, reviews *store.ReviewStore, ref models.ReviewRef) string {
	t.Helper()
	build, err := reviews.PageBuild(ref)
	if err != nil || build == "" {
		t.Fatalf("no page for %s: %v", ref.ID(), err)
	}
	return build
}

func filesUnder(t *testing.T, folder, as string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(folder, path)
			out = append(out, as+"/"+filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return out
}

func whereItDiffers(was, now string) string {
	var a, b any
	if json.Unmarshal([]byte(was), &a) != nil || json.Unmarshal([]byte(now), &b) != nil {
		return ""
	}
	return "\n" + firstDifference("", a, b)
}

func firstDifference(at string, a, b any) string {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok {
			break
		}
		keys := make([]string, 0, len(x)+len(y))
		for k := range x {
			keys = append(keys, k)
		}
		for k := range y {
			if _, ok := x[k]; !ok {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !reflect.DeepEqual(x[k], y[k]) {
				return firstDifference(at+"."+k, x[k], y[k])
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
	return fmt.Sprintf("  %s:\n    kept %v\n    Go gives %v", at, a, b)
}

func plainJSON(v any) (string, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(v)
	return strings.TrimSuffix(out.String(), "\n"), err
}

func spaced(t *testing.T, text string) string {
	t.Helper()
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(text), "", " "); err != nil {
		t.Fatal(err)
	}
	return out.String() + "\n"
}

func noneEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
