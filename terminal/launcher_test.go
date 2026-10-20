package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const standInProgram = `package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("ran " + strings.Join(os.Args[1:], " "))
	if len(os.Args) > 1 && os.Args[1] == "fail" {
		os.Exit(3)
	}
}
`

func launcherTree(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	script, err := os.ReadFile(filepath.Join("..", "bin", "lazyreview"))
	if err != nil {
		t.Fatal(err)
	}
	for path, text := range map[string]string{
		"bin/lazyreview":          string(script),
		"terminal/go.mod":         "module example.com/standin\n\ngo 1.24\n",
		"terminal/main.go":        standInProgram,
		"terminal/main_test.go":   "package main\n",
		"terminal/testdata/x.go":  "package testdata\n",
		"terminal/vendor/note.go": "package vendor\n",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func launch(t *testing.T, script string, env []string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(script, args...)
	cmd.Env = env
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return out.String(), errOut.String(), code
}

func later(t *testing.T, path string, minutes int) {
	t.Helper()
	future := time.Now().Add(time.Duration(minutes) * time.Minute)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
}

func TestTheLauncherBuildsTheProgramWhenItsCodeChangedThenRunsIt(t *testing.T) {
	root := launcherTree(t)
	script := filepath.Join(root, "bin", "lazyreview")
	env := os.Environ()

	out, said, code := launch(t, script, env, "load", "a b")
	if code != 0 || out != "ran load a b\n" || said != "building lazyreview…\n" {
		t.Fatalf("the first run printed %q and said %q, exit %d", out, said, code)
	}
	if _, err := os.Stat(filepath.Join(root, "terminal", "bin", "lazyreview")); err != nil {
		t.Fatal("the program was not kept in terminal/bin")
	}
	if out, said, _ := launch(t, script, env, "list"); out != "ran list\n" || said != "" {
		t.Fatalf("a run with nothing changed printed %q and said %q", out, said)
	}
	for _, path := range []string{"terminal/main_test.go", "terminal/testdata/x.go", "terminal/vendor/note.go"} {
		later(t, filepath.Join(root, path), 1)
		if _, said, _ := launch(t, script, env); said != "" {
			t.Fatalf("a change to %s built the program again", path)
		}
	}
	for i, path := range []string{"terminal/main.go", "terminal/go.mod"} {
		later(t, filepath.Join(root, path), 2*i+2)
		if _, said, _ := launch(t, script, env); said != "building lazyreview…\n" {
			t.Fatalf("a change to %s did not build the program again: %q", path, said)
		}
		later(t, filepath.Join(root, "terminal", "bin", "lazyreview"), 2*i+3)
	}

	link := filepath.Join(t.TempDir(), "lazyreview")
	if err := os.Symlink(script, link); err != nil {
		t.Fatal(err)
	}
	if out, _, code := launch(t, link, env, "fail"); out != "ran fail\n" || code != 3 {
		t.Fatalf("through a link, it printed %q and exited %d; want the program's own exit code", out, code)
	}
}

func TestTheLauncherSaysWhenItCannotBuild(t *testing.T) {
	root := launcherTree(t)
	script := filepath.Join(root, "bin", "lazyreview")
	if err := os.WriteFile(filepath.Join(root, "terminal", "main.go"), []byte("package main\n\nfunc main() { nope() }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, said, code := launch(t, script, os.Environ())
	if code != 1 || !strings.Contains(said, "lazyreview: could not build lazyreview: ") || !strings.Contains(said, "undefined: nope") {
		t.Fatalf("a program that does not build says %q, exit %d", said, code)
	}
	if _, err := os.Stat(filepath.Join(root, "terminal", "bin", "lazyreview")); !os.IsNotExist(err) {
		t.Fatal("a failed build left a program behind")
	}

	noGo := t.TempDir()
	for _, tool := range []string{"sh", "dirname", "readlink", "find", "head"} {
		path, err := exec.LookPath(tool)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(path, filepath.Join(noGo, tool)); err != nil {
			t.Fatal(err)
		}
	}
	_, said, code = launch(t, script, []string{"PATH=" + noGo, "HOME=" + os.Getenv("HOME")})
	if code != 1 || said != "lazyreview: lazyreview is built with Go 1.24 or later, and go is not on the PATH; install it from https://go.dev/dl\n" {
		t.Fatalf("with no go on the PATH, it says %q, exit %d", said, code)
	}
}
