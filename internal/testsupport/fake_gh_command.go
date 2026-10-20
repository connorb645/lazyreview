package testsupport

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const fakeGhRoutes = "LAZYREVIEW_FAKE_GH"

// FakeGhCommand is the stand-in gh as a command, for tests that run
// lazyreview as a program: the test program itself, on the PATH as gh,
// answering from routes as FakeGh does. Env is what a process needs to run
// it as gh. A package whose tests run it calls ActAsFakeGh first in its
// TestMain.
type FakeGhCommand struct {
	Env    []string
	routes string
	t      *testing.T
}

// NewFakeGhCommand makes the stand-in, answering from routes.
func NewFakeGhCommand(t *testing.T, routes Routes) *FakeGhCommand {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	folder := t.TempDir()
	if err := os.Symlink(self, filepath.Join(folder, "gh")); err != nil {
		t.Fatal(err)
	}
	text, err := json.Marshal(routes)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(folder, "routes.json")
	if err := os.WriteFile(path, text, 0o644); err != nil {
		t.Fatal(err)
	}
	env := []string{"PATH=" + folder + string(os.PathListSeparator) + os.Getenv("PATH"), fakeGhRoutes + "=" + path}
	return &FakeGhCommand{Env: env, routes: path, t: t}
}

// Calls is every call the stand-in was asked to make so far.
func (f *FakeGhCommand) Calls() []Call {
	f.t.Helper()
	text, err := os.ReadFile(f.routes + ".calls")
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		f.t.Fatal(err)
	}
	var calls []Call
	for _, line := range strings.Split(strings.TrimSpace(string(text)), "\n") {
		var call Call
		if err := json.Unmarshal([]byte(line), &call); err != nil {
			f.t.Fatal(err)
		}
		calls = append(calls, call)
	}
	return calls
}

// ActAsFakeGh is the stand-in at work when a test ran this program as gh,
// and returns at once when none did.
func ActAsFakeGh() {
	routes := os.Getenv(fakeGhRoutes)
	if routes == "" || filepath.Base(os.Args[0]) != "gh" {
		return
	}
	os.Exit(fakeGhCommand(routes, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func fakeGhCommand(routes string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	text, err := os.ReadFile(routes)
	f := &FakeGh{}
	if err == nil {
		err = json.Unmarshal(text, &f.Routes)
	}
	if err != nil {
		_, _ = io.WriteString(stderr, "the fake gh could not read its routes: "+err.Error()+"\n")
		return 1
	}
	var input []byte
	if slices.Contains(args, "--input") {
		input, _ = io.ReadAll(stdin)
	}
	ran := f.Run(args, input, os.Environ())
	for _, call := range f.Calls {
		logged, _ := json.Marshal(call)
		appendLine(routes+".calls", string(logged))
	}
	_, _ = io.WriteString(stdout, ran.Stdout)
	_, _ = io.WriteString(stderr, ran.Stderr)
	return ran.Code
}
