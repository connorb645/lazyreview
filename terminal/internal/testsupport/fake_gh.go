// Package testsupport holds what the tests of several packages share:
// stand-ins for the gh command and for Claude Code, and a copy of the
// stand-in data home the terminal app's recorded answers come from.
package testsupport

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/commands/github"
)

var (
	fakeReviewRoute = regexp.MustCompile(`^repos/([^/]+/[^/]+)/pulls/(\d+)/reviews$`)
	fakeHunk        = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)
)

// FakeGh answers gh's calls from routes, as the Node tests' fake gh does:
// "GET <path>" or "POST <path>", led by "token-<login> " for one account's
// calls; an answer {"$fail": text} fails. Like GitHub, it refuses a review
// whose comment is not on a line of its pull request's diff, as the files
// route gives it, or whose range leaves one hunk or ends before it starts.
type FakeGh struct {
	Routes map[string]json.RawMessage
	Calls  []Call
}

// FakeGhFrom answers from the routes in the file at path.
func FakeGhFrom(t *testing.T, path string) *FakeGh {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read the routes: %v", err)
	}
	f := &FakeGh{}
	if err := json.Unmarshal(text, &f.Routes); err != nil {
		t.Fatalf("could not read the routes: %v", err)
	}
	return f
}

// Route answers key, such as "GET user", with answer, as JSON.
func (f *FakeGh) Route(t *testing.T, key string, answer any) {
	t.Helper()
	text, err := json.Marshal(answer)
	if err != nil {
		t.Fatal(err)
	}
	f.Routes[key] = text
}

// Change changes the answer of key, as JSON, with change.
func (f *FakeGh) Change(t *testing.T, key string, change func(map[string]any)) {
	t.Helper()
	var answer map[string]any
	if err := json.Unmarshal(f.Routes[key], &answer); err != nil {
		t.Fatalf("%s: %v", key, err)
	}
	change(answer)
	f.Route(t, key, answer)
}

// Posts are the reviews gh was asked to create.
func (f *FakeGh) Posts() []Call {
	var out []Call
	for _, c := range f.Calls {
		if c.Method == "POST" && strings.HasSuffix(c.Path, "/reviews") {
			out = append(out, c)
		}
	}
	return out
}

// Run runs gh as github.Run does.
func (f *FakeGh) Run(args []string, input []byte, env []string) github.Ran {
	if len(args) > 2 && args[0] == "auth" && args[1] == "token" {
		return github.Ran{Stdout: "token-" + args[len(args)-1] + "\n"}
	}
	method, path := "GET", ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-X":
			method = args[i+1]
			i++
		case "--hostname":
			i++
		case "--input":
			if method == "GET" {
				method = "POST"
			}
			i++
		default:
			if path == "" {
				path = args[i]
			}
		}
	}
	token := ""
	for _, e := range env {
		if v, ok := strings.CutPrefix(e, "GH_TOKEN="); ok {
			token = v
		}
	}
	key := method + " " + path
	f.Calls = append(f.Calls, Call{Method: method, Path: path, Body: input, Token: token})
	if m := fakeReviewRoute.FindStringSubmatch(path); method == "POST" && m != nil {
		if why := f.refusal(m[1], m[2], input); why != "" {
			return github.Ran{Code: 1, Stderr: "gh: Unprocessable Entity (HTTP 422)\n" + why + "\n"}
		}
	}
	for _, k := range []string{token + " " + key, token + " *", key} {
		answer, ok := f.Routes[k]
		if !ok {
			continue
		}
		var fail struct {
			Fail *string `json:"$fail"`
		}
		if json.Unmarshal(answer, &fail) == nil && fail.Fail != nil {
			return github.Ran{Code: 1, Stderr: *fail.Fail + "\n"}
		}
		return github.Ran{Stdout: string(answer)}
	}
	return github.Ran{Code: 1, Stderr: "HTTP 404: Not Found (fake gh has no " + key + ")\n"}
}

func fakePlaces(patch string) map[string]map[int]fakePlace {
	at := map[string]map[int]fakePlace{"RIGHT": {}, "LEFT": {}}
	hunk, old, now := -1, 0, 0
	for i, row := range strings.Split(patch, "\n") {
		if m := fakeHunk.FindStringSubmatch(row); m != nil {
			hunk++
			old, _ = strconv.Atoi(m[1])
			now, _ = strconv.Atoi(m[2])
			continue
		}
		if hunk < 0 || row == "" || row[0] == '\\' {
			continue
		}
		if row[0] != '-' {
			at["RIGHT"][now] = fakePlace{hunk, i}
			now++
		}
		if row[0] != '+' {
			at["LEFT"][old] = fakePlace{hunk, i}
			old++
		}
	}
	return at
}

func (f *FakeGh) refusal(repo, number string, input []byte) string {
	var body struct {
		Comments []struct {
			Path      string `json:"path"`
			Line      int    `json:"line"`
			Side      string `json:"side"`
			StartLine int    `json:"start_line"`
			StartSide string `json:"start_side"`
		} `json:"comments"`
	}
	_ = json.Unmarshal(input, &body)
	var files []struct {
		Filename string `json:"filename"`
		Patch    string `json:"patch"`
	}
	_ = json.Unmarshal(f.Routes["GET repos/"+repo+"/pulls/"+number+"/files?per_page=100&page=1"], &files)
	var refused []string
	for _, c := range body.Comments {
		patch, found := "", false
		for _, file := range files {
			if file.Filename == c.Path {
				patch, found = file.Patch, true
			}
		}
		if !found {
			refused = append(refused, c.Path+": Path could not be resolved")
			continue
		}
		at := fakePlaces(patch)
		end, ok := at[c.Side][c.Line]
		if !ok {
			refused = append(refused, c.Path+":"+strconv.Itoa(c.Line)+": Line could not be resolved")
			continue
		}
		if c.StartLine == 0 {
			continue
		}
		start, ok := at[c.StartSide][c.StartLine]
		switch {
		case !ok:
			refused = append(refused, c.Path+":"+strconv.Itoa(c.StartLine)+": Start line could not be resolved")
		case start.hunk != end.hunk:
			refused = append(refused, c.Path+":"+strconv.Itoa(c.StartLine)+"-"+strconv.Itoa(c.Line)+": start_line must be part of the same hunk as the line")
		case start.row >= end.row:
			refused = append(refused, c.Path+":"+strconv.Itoa(c.StartLine)+"-"+strconv.Itoa(c.Line)+": start_line must precede the end line")
		}
	}
	return strings.Join(refused, "\n")
}

// Testdata is a path in the terminal app's testdata folder, from a package
// two folders below it.
func Testdata(parts ...string) string {
	return filepath.Join(append([]string{"..", "..", "testdata"}, parts...)...)
}

// StandInHome is a copy of the data home the recorded answers came from, in
// a folder of the test's own: every review, or only those of reviews when
// any are named, by id.
func StandInHome(t *testing.T, reviews ...string) string {
	t.Helper()
	from, to := Testdata("home"), t.TempDir()
	err := filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(from, path)
		if len(reviews) > 0 && !slices.ContainsFunc(reviews, func(id string) bool {
			return strings.HasPrefix(filepath.ToSlash(rel), "reviews/"+id+"/")
		}) {
			return nil
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Join(to, rel)), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(to, rel), text, 0o644)
	})
	if err != nil {
		t.Fatalf("could not copy the stand-in data home: %v", err)
	}
	return to
}
