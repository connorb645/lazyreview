package land_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/land"
	"github.com/connorb645/lazyreview/terminal/internal/testsupport"
)

func TestABranchLandsOnMainOnEveryRemoteAndHereAfterSayingWhatItBringsAndWhatItWillDo(t *testing.T) {
	t.Parallel()
	repo := newLandingRepo(t)
	feature := repo.published()
	short := repo.short(feature)
	landing := land.NewLanding(repo.Work, nil)
	plan, err := landing.Plan("feature")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Landing feature at " + short + " brings 1 commit to main:",
		"  " + short + " Add a",
		"It will:",
		"  push " + short + " to main on mirror, without force",
		"  push " + short + " to main on origin, without force",
		"  move main here to " + short + ", fast-forward only",
	}
	if got := land.Describe(plan); !slices.Equal(got, want) {
		t.Fatalf("described %q, want %q", got, want)
	}
	landed, err := landing.Land(plan)
	if err != nil {
		t.Fatal(err)
	}
	if want := "main is at " + short + " on mirror, origin and here."; landed != want {
		t.Fatalf("landed %q, want %q", landed, want)
	}
	if got := repo.mains(); !slices.Equal(got, []string{feature, feature, feature}) {
		t.Fatalf("main is at %q, want %s everywhere", got, feature)
	}
	if text, err := os.ReadFile(filepath.Join(repo.Work, "a.txt")); err != nil || string(text) != "a\n" {
		t.Fatalf("a.txt reads %q, %v", text, err)
	}
	if status := repo.Git("status", "--porcelain"); status != "" {
		t.Fatalf("the checkout has changes: %q", status)
	}
}

func TestAStackLandsByItsTopBranchBringingEveryBranchBelowIt(t *testing.T) {
	t.Parallel()
	repo := newLandingRepo(t)
	bottom := repo.Commit("stack-bottom", map[string]string{"a.txt": "a\n"}, "Add a")
	top := repo.Commit("stack-top", map[string]string{"b.txt": "b\n"}, "Add b")
	repo.publish()
	repo.Checkout("main")
	landing := land.NewLanding(repo.Work, nil)
	plan, err := landing.Plan("stack-top")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Landing stack-top at " + repo.short(top) + " brings 2 commits to main:",
		"  " + repo.short(top) + " Add b",
		"  " + repo.short(bottom) + " Add a",
	}
	if got := land.Describe(plan)[:3]; !slices.Equal(got, want) {
		t.Fatalf("described %q, want %q", got, want)
	}
	if _, err := landing.Land(plan); err != nil {
		t.Fatal(err)
	}
	if got := repo.mains(); !slices.Equal(got, []string{top, top, top}) {
		t.Fatalf("main is at %q, want %s everywhere", got, top)
	}
	repo.Git("merge-base", "--is-ancestor", bottom, "main")
}

func TestItRefusesWithOnePlainLineAndChangesNothing(t *testing.T) {
	t.Parallel()
	refusals := []struct {
		when  string
		setup func(t *testing.T, repo *landingRepo) (branch, says string)
	}{
		{"no branch is given", func(t *testing.T, repo *landingRepo) (string, string) {
			repo.published()
			return "", exactly("say which branch to land: make land BRANCH=<branch>")
		}},
		{"the checkout is not on main", func(t *testing.T, repo *landingRepo) (string, string) {
			repo.published()
			repo.Checkout("feature")
			return "feature", exactly("this checkout is on feature, not main")
		}},
		{"the checkout is on no branch", func(t *testing.T, repo *landingRepo) (string, string) {
			repo.published()
			repo.Git("checkout", "-q", "--detach", "main")
			return "feature", exactly("this checkout is on no branch, not main")
		}},
		{"the checkout has uncommitted changes", func(t *testing.T, repo *landingRepo) (string, string) {
			repo.published()
			if err := os.WriteFile(filepath.Join(repo.Work, "notes.txt"), []byte("later\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return "feature", exactly("this checkout has uncommitted changes")
		}},
		{"there is no remote", func(t *testing.T, repo *landingRepo) (string, string) {
			repo.published()
			repo.Git("remote", "remove", "mirror")
			repo.Git("remote", "remove", "origin")
			return "feature", exactly("this repository has no remote to land on")
		}},
		{"a remote cannot be fetched", func(t *testing.T, repo *landingRepo) (string, string) {
			repo.published()
			repo.Git("remote", "set-url", "mirror", filepath.Join(filepath.Dir(repo.origin), "gone.git"))
			return "feature", `^could not fetch mirror: '.*gone\.git' does not appear to be a git repository$`
		}},
		{"a remote does not have the branch", func(t *testing.T, repo *landingRepo) (string, string) {
			repo.Commit("feature", map[string]string{"a.txt": "a\n"}, "Add a")
			repo.Git("push", "-q", "origin", "feature")
			repo.Checkout("main")
			return "feature", exactly("feature is not on mirror")
		}},
		{"two remotes have the branch at different commits", func(t *testing.T, repo *landingRepo) (string, string) {
			one := repo.Commit("feature", map[string]string{"a.txt": "one\n"}, "Add a")
			repo.Git("push", "-q", "mirror", "feature")
			repo.Checkout("main")
			repo.Git("branch", "-q", "-D", "feature")
			two := repo.Commit("feature", map[string]string{"a.txt": "two\n"}, "Add a")
			repo.Git("push", "-q", "origin", "feature")
			repo.Checkout("main")
			return "feature", exactly("feature is at different commits: " + repo.short(one) + " on mirror, " + repo.short(two) + " on origin")
		}},
		{"a remote has no main", func(t *testing.T, repo *landingRepo) (string, string) {
			repo.published()
			testsupport.RunGit(t, repo.origin, "update-ref", "-d", "refs/heads/main")
			return "feature", exactly("there is no main on origin")
		}},
		{"main on a remote has commits the branch lacks", func(t *testing.T, repo *landingRepo) (string, string) {
			repo.published()
			repo.Commit("elsewhere", map[string]string{"b.txt": "b\n"}, "Add b")
			repo.Git("push", "-q", "origin", "elsewhere:main")
			repo.Checkout("main")
			return "feature", exactly("main on origin has commits feature lacks, so landing it would not be a fast-forward")
		}},
		{"main here has commits the branch lacks", func(t *testing.T, repo *landingRepo) (string, string) {
			repo.published()
			repo.Commit("main", map[string]string{"b.txt": "b\n"}, "Add b")
			return "feature", exactly("main here has commits feature lacks, so landing it would not be a fast-forward")
		}},
		{"a remote will not take a push", func(t *testing.T, repo *landingRepo) (string, string) {
			repo.published()
			repo.Git("remote", "set-url", "--push", "origin", filepath.Join(filepath.Dir(repo.origin), "gone.git"))
			return "feature", `^origin would not take the push: '.*gone\.git' does not appear to be a git repository$`
		}},
	}
	for _, c := range refusals {
		t.Run(c.when, func(t *testing.T) {
			t.Parallel()
			repo := newLandingRepo(t)
			branch, says := c.setup(t, repo)
			before := repo.state()
			_, err := land.NewLanding(repo.Work, nil).Plan(branch)
			if err == nil || !regexp.MustCompile(says).MatchString(err.Error()) {
				t.Fatalf("refused with %v, want %s", err, says)
			}
			if after := repo.state(); !slices.Equal(after, before) {
				t.Fatalf("the refusal changed\n%q\nto\n%q", before, after)
			}
		})
	}
}

func TestAPushTheSecondRemoteRefusesLeavesTheRemotesApartAndSaysHowToFinishWhichRunningItAgainDoes(t *testing.T) {
	t.Parallel()
	repo := newLandingRepo(t)
	feature := repo.published()
	short := repo.short(feature)
	repo.refuse("origin")
	landing := land.NewLanding(repo.Work, nil)
	plan, err := landing.Plan("feature")
	if err != nil {
		t.Fatal(err)
	}
	_, err = landing.Land(plan)
	want := strings.Join([]string{
		"origin refused the push: pre-receive hook declined, saying: pushes are closed",
		"the remotes are apart: main is at " + short + " on mirror, but still at " + repo.short(repo.base) + " on origin",
		"fix that, then finish with: make land BRANCH=feature",
	}, "\n")
	if err == nil || err.Error() != want {
		t.Fatalf("refused with %v, want %q", err, want)
	}
	if got := repo.mains(); !slices.Equal(got, []string{feature, repo.base, repo.base}) {
		t.Fatalf("main is at %q: the push to mirror stays, and main here waits", got)
	}

	repo.allow("origin")
	again, err := landing.Plan("feature")
	if err != nil {
		t.Fatal(err)
	}
	steps := []string{
		"It will:",
		"  push " + short + " to main on origin, without force",
		"  move main here to " + short + ", fast-forward only",
	}
	if got := land.Describe(again)[2:]; !slices.Equal(got, steps) {
		t.Fatalf("described %q, want %q", got, steps)
	}
	landed, err := landing.Land(again)
	if err != nil {
		t.Fatal(err)
	}
	if want := "main is at " + short + " on mirror, origin and here."; landed != want {
		t.Fatalf("landed %q, want %q", landed, want)
	}
	if got := repo.mains(); !slices.Equal(got, []string{feature, feature, feature}) {
		t.Fatalf("main is at %q, want %s everywhere", got, feature)
	}
}

func TestAPushTheFirstRemoteRefusesChangesNothing(t *testing.T) {
	t.Parallel()
	repo := newLandingRepo(t)
	repo.published()
	repo.refuse("mirror")
	landing := land.NewLanding(repo.Work, nil)
	before := repo.state()
	plan, err := landing.Plan("feature")
	if err != nil {
		t.Fatal(err)
	}
	_, err = landing.Land(plan)
	if want := "mirror refused the push, so nothing has changed: pre-receive hook declined, saying: pushes are closed"; err == nil || err.Error() != want {
		t.Fatalf("refused with %v, want %q", err, want)
	}
	if after := repo.state(); !slices.Equal(after, before) {
		t.Fatalf("the refusal changed\n%q\nto\n%q", before, after)
	}
}

func TestWithMainAtTheBranchEverywhereAlreadyNothingIsPushedOrMoved(t *testing.T) {
	t.Parallel()
	repo := newLandingRepo(t)
	feature := repo.published()
	short := repo.short(feature)
	landing := land.NewLanding(repo.Work, nil)
	first, err := landing.Plan("feature")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := landing.Land(first); err != nil {
		t.Fatal(err)
	}
	plan, err := landing.Plan("feature")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := land.Describe(plan), []string{"Landing feature at " + short + ", which main here is at already."}; !slices.Equal(got, want) {
		t.Fatalf("described %q, want %q", got, want)
	}
	landed, err := landing.Land(plan)
	if err != nil {
		t.Fatal(err)
	}
	if want := "main is at " + short + " on mirror, origin and here."; landed != want {
		t.Fatalf("landed %q, want %q", landed, want)
	}
}

func exactly(text string) string {
	return "^" + regexp.QuoteMeta(text) + "$"
}
