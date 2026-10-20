package cli

import (
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
)

func TestTheAddressLoadAndListPrintNamesItsReviewOnAnyPortWhateverFollowsIt(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for _, address := range []string{
		"lazyreview://github.com/octo/widgets/30",
		"lazyreview://github.com/octo/widgets/30/",
		"http://127.0.0.1:4747/review/github.com/octo/widgets/30",
		"http://127.0.0.1:48231/review/github.com/octo/widgets/30#review",
		"http://localhost:4747/review/github.com/octo/widgets/30/?file=a.swift",
		"https://127.0.0.1/review/github.com/octo/widgets/30",
		"github.com/octo/widgets/30",
	} {
		link, err := ReadTerminalLink(tr, address)
		if err != nil || link.Review == nil || link.Review.ID() != "github.com/octo/widgets/30" {
			t.Errorf("%s names %+v, %v", address, link, err)
		}
	}
	for text, id := range map[string]string{
		"lazyreview://local/widgets-1a2b3c4d/feature~search":                 "local/widgets-1a2b3c4d/feature~search",
		"http://127.0.0.1:4747/review/local/widgets-1a2b3c4d/feature~search": "local/widgets-1a2b3c4d/feature~search",
		"local/widgets-1a2b3c4d/feature~search":                              "local/widgets-1a2b3c4d/feature~search",
	} {
		if link, err := ReadTerminalLink(tr, text); err != nil || link.Review == nil || link.Review.ID() != id {
			t.Errorf("%s names %+v, %v", text, link, err)
		}
	}
	link, err := ReadTerminalLink(tr, "http://127.0.0.1:4747/review/local/widgets-1a2b3c4d/fix%2525")
	if branch, ok := link.Review.(models.BranchRef); err != nil || !ok || branch.Branch != "fix%25" {
		t.Errorf("an address is decoded once, as the app decodes it: %+v, %v", link, err)
	}
}

func TestAPullRequestLinkNamesThePullRequest(t *testing.T) {
	link, err := ReadTerminalLink(i18n.EnglishTranslationSet(), "https://github.com/octo/widgets/pull/30/files")
	if err != nil || link.Review != nil || link.Pull == nil || link.Pull.ID() != "github.com/octo/widgets/30" {
		t.Fatalf("a pull request link names %+v, %v", link, err)
	}
}

func TestTextThatNamesNoReviewOrPullRequestIsRefusedInOneLineNamingAllOfIt(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for _, text := range []string{
		"widgets",
		"http://127.0.0.1:4747/",
		"http://127.0.0.1:4747/review/garbage",
		"http://127.0.0.1:4747/review/github.com/octo/widgets/%E0",
		"http://127.0.0.1:4747/review/local/widgets-1a2b3c4d/%E0",
		"http://127.0.0.1:4747/api/reviews/github.com/octo/widgets/30",
		"https://example.com/octo/widgets",
		"lazyreview://",
	} {
		_, err := ReadTerminalLink(tr, text)
		want := `"` + text + `" is not the address of a review or of a pull request; lazyreview list shows every review`
		if err == nil || err.Error() != want {
			t.Errorf("%s says %v", text, err)
		}
	}
}

func TestAReviewsAddressEncodesEachPartOfItsID(t *testing.T) {
	for ref, want := range map[models.ReviewRef]string{
		models.PullRef{Host: "github.com", Repo: "octo/widgets", Pull: 12}: "lazyreview://github.com/octo/widgets/12",
		models.BranchRef{RepoKey: "work-1a2b3c4d", Branch: "fix/a+b@c#d"}:  "lazyreview://local/work-1a2b3c4d/fix~a%2Bb%40c%23d",
		models.BranchRef{RepoKey: "work-1a2b3c4d", Branch: "café(1)!"}:     "lazyreview://local/work-1a2b3c4d/caf%C3%A9(1)!",
		models.BranchRef{RepoKey: "work-1a2b3c4d", Branch: "percent%25"}:   "lazyreview://local/work-1a2b3c4d/percent%2525",
	} {
		if got := ReviewAddress(ref); got != want {
			t.Errorf("%s has the address %s, want %s", ref.ID(), got, want)
		}
		if link, err := ReadTerminalLink(i18n.EnglishTranslationSet(), ReviewAddress(ref)); err != nil || link.Review.ID() != ref.ID() {
			t.Errorf("%s opens %+v, %v", ReviewAddress(ref), link, err)
		}
	}
}

func TestTheCommandLineIsReadOptionsInAnyOrder(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	o, err := ParseArguments(tr, []string{"load", "--account", "lee", "https://github.com/octo/widgets/pull/21", "--context", "https://github.com/octo/widgets/pull/22", "--config", "c.json"})
	if err != nil || o.Command != "load" || *o.Account != "lee" || *o.Config != "c.json" || !o.Context || o.NoContext || strings.Join(o.Positional, " ") != "https://github.com/octo/widgets/pull/21 https://github.com/octo/widgets/pull/22" {
		t.Fatalf("load reads as %+v, %v", o, err)
	}
	o, err = ParseArguments(tr, []string{"load", "--repo", ".", "--base", "main", "--head", "fix/x", "--title", "Fix", "--no-context"})
	if err != nil || *o.Repo != "." || *o.Base != "main" || *o.Head != "fix/x" || *o.Title != "Fix" || !o.NoContext {
		t.Fatalf("a branch's load reads as %+v, %v", o, err)
	}
	for _, argv := range [][]string{nil, {"help"}, {"--help"}, {"-h"}, {"load", "x", "--help"}, {"list", "-h"}} {
		if o, err := ParseArguments(tr, argv); err != nil || o.Command != "help" {
			t.Errorf("%v reads as %+v, %v; want help", argv, o, err)
		}
	}
	if o, err := ParseArguments(tr, []string{"config", "move", "--check"}); err != nil || !o.Check || o.Positional[0] != "move" {
		t.Fatalf("config move --check reads as %+v, %v", o, err)
	}
}

func TestTheCommandLineNamesWhatWasWrongThenGivesTheUsage(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for want, argv := range map[string][]string{
		"load needs a pull request, or --repo with --base and --head": {"load"},
		"--repo needs --head":                          {"load", "--repo", ".", "--base", "main"},
		"--repo needs --base":                          {"load", "--repo", ".", "--head", "b"},
		"--head needs --repo":                          {"load", "--head", "feature"},
		"--title needs --repo":                         {"load", "x", "--title", "T"},
		"load takes pull requests or --repo, not both": {"load", "octo/widgets#1", "--repo", ".", "--base", "a", "--head", "b"},
		"--account is for pull requests; a branch loads from its own repository": {"load", "--repo", ".", "--base", "a", "--head", "b", "--account", "me"},
		"--account needs a value":                                       {"load", "--account", "--context", "x"},
		"--config needs a value":                                        {"load", "x", "--config"},
		"terminal takes 1 argument":                                     {"terminal", "octo/widgets#1", "octo/widgets#2"},
		"list takes no arguments":                                       {"list", "x"},
		"attach needs 2 arguments":                                      {"attach", "x"},
		"attach takes 2 arguments":                                      {"attach", "x", "y", "z"},
		`unknown command "frobnicate"`:                                  {"frobnicate"},
		`unknown command "serve"`:                                       {"serve"},
		`unknown command "-x"`:                                          {"-x"},
		"unknown option --colour for load":                              {"load", "x", "--colour"},
		"config needs one of path, defaults, check or move":             {"config"},
		`config needs one of path, defaults, check or move, not "frob"`: {"config", "frob"},
		"--check is for config move":                                    {"config", "path", "--check"},
		"config takes 1 argument":                                       {"config", "path", "more"},
	} {
		_, err := ParseArguments(tr, argv)
		if err == nil || err.Error() != want+"\n\n"+tr.Usage {
			t.Errorf("%v says %v, want %q and the usage", argv, err, want)
		}
	}
}
