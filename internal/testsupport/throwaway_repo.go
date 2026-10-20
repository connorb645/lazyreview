package testsupport

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ThrowawayRepo is a git repository for a test, in a folder of its own that
// goes when the test ends. Every commit is made at one time, so the same
// files and messages always give the same commits.
type ThrowawayRepo struct {
	Work string
	t    *testing.T
}

// NewThrowawayRepo makes a repository whose first branch is main.
func NewThrowawayRepo(t *testing.T) *ThrowawayRepo {
	t.Helper()
	r := &ThrowawayRepo{Work: filepath.Join(t.TempDir(), "work"), t: t}
	if err := os.Mkdir(r.Work, 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("init", "-q", "-b", "main")
	return r
}

// Commit writes files, by path, on branch, made from the branch checked out
// when it is new, and commits them, returning the commit.
func (r *ThrowawayRepo) Commit(branch string, files map[string]string, message string) string {
	r.t.Helper()
	if r.Git("symbolic-ref", "--short", "HEAD") != branch {
		if r.Git("branch", "--list", branch) != "" {
			r.Git("checkout", "-q", branch)
		} else {
			r.Git("checkout", "-q", "-b", branch)
		}
	}
	for path, text := range files {
		full := filepath.Join(r.Work, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			r.t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			r.t.Fatal(err)
		}
	}
	r.Git("add", "-A")
	r.Git("commit", "-q", "--allow-empty", "-m", message)
	return r.Git("rev-parse", "HEAD")
}

// Checkout checks branch out.
func (r *ThrowawayRepo) Checkout(branch string) {
	r.Git("checkout", "-q", branch)
}

// Git runs git in the repository and returns what it printed, trimmed.
func (r *ThrowawayRepo) Git(args ...string) string {
	r.t.Helper()
	return RunGit(r.t, r.Work, args...)
}

// RunGit runs git in dir as a test's own user, at the fixed time, and
// returns what it printed, trimmed.
func RunGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE=2026-10-15T12:00:00Z", "GIT_COMMITTER_DATE=2026-10-15T12:00:00Z")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}
