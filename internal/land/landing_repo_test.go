package land_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/connorb645/lazyreview/internal/testsupport"
)

type landingRepo struct {
	*testsupport.ThrowawayRepo
	t      *testing.T
	mirror string
	origin string
	base   string
}

func TestMain(m *testing.M) {
	os.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	os.Exit(m.Run())
}

func newLandingRepo(t *testing.T) *landingRepo {
	t.Helper()
	remotes := t.TempDir()
	r := &landingRepo{
		ThrowawayRepo: testsupport.NewThrowawayRepo(t),
		t:             t,
		mirror:        filepath.Join(remotes, "mirror.git"),
		origin:        filepath.Join(remotes, "origin.git"),
	}
	for _, bare := range []string{r.mirror, r.origin} {
		testsupport.RunGit(t, remotes, "init", "-q", "--bare", "-b", "main", bare)
	}
	r.Git("remote", "add", "mirror", r.mirror)
	r.Git("remote", "add", "origin", r.origin)
	r.base = r.Commit("main", map[string]string{"README.md": "widgets\n"}, "Start")
	r.publish()
	return r
}

func (r *landingRepo) publish() {
	r.t.Helper()
	for _, remote := range []string{"mirror", "origin"} {
		r.Git("push", "-q", remote, "--all")
	}
}

func (r *landingRepo) published() string {
	r.t.Helper()
	feature := r.Commit("feature", map[string]string{"a.txt": "a\n"}, "Add a")
	r.publish()
	r.Checkout("main")
	return feature
}

func (r *landingRepo) short(commit string) string {
	r.t.Helper()
	return r.Git("rev-parse", "--short", commit)
}

func (r *landingRepo) bare(remote string) string {
	if remote == "mirror" {
		return r.mirror
	}
	return r.origin
}

func (r *landingRepo) mains() []string {
	r.t.Helper()
	return []string{
		testsupport.RunGit(r.t, r.mirror, "rev-parse", "main"),
		testsupport.RunGit(r.t, r.origin, "rev-parse", "main"),
		r.Git("rev-parse", "main"),
	}
}

func (r *landingRepo) state() []string {
	r.t.Helper()
	return []string{
		r.Git("rev-parse", "--symbolic-full-name", "HEAD", "HEAD"),
		r.Git("for-each-ref", "refs/heads"),
		testsupport.RunGit(r.t, r.mirror, "for-each-ref"),
		testsupport.RunGit(r.t, r.origin, "for-each-ref"),
	}
}

func (r *landingRepo) hook(remote string) string {
	return filepath.Join(r.bare(remote), "hooks", "pre-receive")
}

func (r *landingRepo) refuse(remote string) {
	r.t.Helper()
	if err := os.WriteFile(r.hook(remote), []byte("#!/bin/sh\necho \"error: pushes are closed\" >&2\nexit 1\n"), 0o755); err != nil {
		r.t.Fatal(err)
	}
}

func (r *landingRepo) allow(remote string) {
	r.t.Helper()
	if err := os.Remove(r.hook(remote)); err != nil {
		r.t.Fatal(err)
	}
}
