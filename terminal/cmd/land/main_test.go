package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/testsupport"
)

func TestMain(m *testing.M) {
	os.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	os.Exit(m.Run())
}

func TestLandSaysWhatItWillDoThenLandsTheBranchOnTheCheckoutItRunsIn(t *testing.T) {
	repo := testsupport.NewThrowawayRepo(t)
	origin := filepath.Join(t.TempDir(), "origin.git")
	testsupport.RunGit(t, filepath.Dir(origin), "init", "-q", "--bare", "-b", "main", origin)
	repo.Git("remote", "add", "origin", origin)
	repo.Commit("main", map[string]string{"README.md": "widgets\n"}, "Start")
	feature := repo.Commit("feature", map[string]string{"a.txt": "a\n"}, "Add a")
	repo.Git("push", "-q", "origin", "--all")
	repo.Checkout("main")
	short := repo.Git("rev-parse", "--short", feature)
	t.Chdir(repo.Work)

	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 1 || stdout.String() != "" || stderr.String() != "land: say which branch to land: make land BRANCH=<branch>\n" {
		t.Fatalf("with no branch: exit %d, printed %q and %q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"feature"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	want := "Landing feature at " + short + " brings 1 commit to main:\n" +
		"  " + short + " Add a\n" +
		"It will:\n" +
		"  push " + short + " to main on origin, without force\n" +
		"  move main here to " + short + ", fast-forward only\n" +
		"main is at " + short + " on origin and here.\n"
	if stdout.String() != want || stderr.String() != "" {
		t.Fatalf("printed %q and %q, want %q", stdout.String(), stderr.String(), want)
	}
	if got := testsupport.RunGit(t, origin, "rev-parse", "main"); got != feature {
		t.Fatalf("main on origin is at %s, want %s", got, feature)
	}
}
