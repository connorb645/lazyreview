package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/config"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/loader"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/store"
)

type standIn struct {
	command *CommandLine
	out     *bytes.Buffer
	home    store.DataHome
	loads   [][]string
	opened  []string
}

func newStandIn(t *testing.T, env map[string]string) *standIn {
	t.Helper()
	tr := i18n.EnglishTranslationSet()
	root := t.TempDir()
	s := &standIn{out: &bytes.Buffer{}, home: store.DataHome{Root: filepath.Join(root, "data")}}
	values := map[string]string{"LAZYREVIEW_HOME": s.home.Root, "LAZYREVIEW_CONFIG_DIR": filepath.Join(root, "config")}
	for name, value := range env {
		values[name] = value
	}
	s.command = &CommandLine{
		tr: tr, out: s.out, getenv: func(name string) string { return values[name] }, platform: runtime.GOOS, home: root,
		load: func(tr *i18n.TranslationSet, dataHome string, pulls loader.PullsRequest, branch *loader.BranchRequest) (loader.Loaded, error) {
			s.loads = append(s.loads, pulls.Links)
			pull, err := models.ParsePullRef(tr, pulls.Links[0])
			if err != nil {
				return loader.Loaded{}, err
			}
			s.keep(t, pull)
			return loader.Loaded{Ref: pull, Notes: []string{"a stand-in note"}}, nil
		},
		launch: func(tr *i18n.TranslationSet, given config.UserConfig, readConfig func() (config.Read, error), review string) error {
			s.opened = append(s.opened, review)
			return nil
		},
	}
	return s
}

func (s *standIn) keep(t *testing.T, pull models.PullRef) {
	t.Helper()
	kept := store.NewReviewStore(s.command.tr, s.home, store.NewLocks(s.command.tr))
	err := kept.Change([]models.ReviewRef{pull}, func() ([]store.FileWrite, error) {
		record, err := kept.RecordWrite(pull, models.Review{ID: pull.ID(), Host: pull.Host, Repo: pull.Repo, Members: []models.ReviewMember{{Pull: pull.Pull}}})
		return []store.FileWrite{record}, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTheTerminalAppOpensTheReviewAnAddressNamesLoadingAPullRequestNoReviewHoldsYet(t *testing.T) {
	s := newStandIn(t, nil)
	if err := s.command.Run([]string{"terminal", "http://127.0.0.1:4747/"}); err == nil || !strings.Contains(err.Error(), "is not the address of a review or of a pull request") {
		t.Fatalf("an address that names no review opens as %v", err)
	}
	for _, address := range []string{
		"https://github.com/octo/widgets/pull/30",
		"lazyreview://github.com/octo/widgets/30",
		"http://127.0.0.1:4747/review/github.com/octo/widgets/30#review",
		"https://github.com/octo/widgets/pull/30",
		"github.com/octo/widgets/30",
	} {
		if err := s.command.Run([]string{"terminal", address}); err != nil {
			t.Fatalf("%s: %v", address, err)
		}
	}
	if want := []string{"https://github.com/octo/widgets/pull/30"}; len(s.loads) != 1 || !slices.Equal(s.loads[0], want) {
		t.Fatalf("the loads were %v; a review already loaded opens as it is", s.loads)
	}
	if said := s.out.String(); said != "loading https://github.com/octo/widgets/pull/30…\nnote: a stand-in note\n" {
		t.Fatalf("it said %q", said)
	}
	if want := slices.Repeat([]string{"github.com/octo/widgets/30"}, 5); !slices.Equal(s.opened, want) {
		t.Fatalf("it opened %v; an address an older lazyreview printed still opens", s.opened)
	}
	if err := s.command.Run([]string{"terminal"}); err != nil || s.opened[len(s.opened)-1] != "" {
		t.Fatalf("with no address, it opens %q, %v; want the home screen", s.opened[len(s.opened)-1], err)
	}
	err := s.command.Run([]string{"terminal", "lazyreview://github.com/octo/widgets/99"})
	if err == nil || err.Error() != "there is no review github.com/octo/widgets/99; lazyreview list shows every review" {
		t.Fatalf("an address of no review opens as %v", err)
	}
}

func TestTheCommandOpensNoDataHomeButOneNamedByItsFullPath(t *testing.T) {
	s := newStandIn(t, map[string]string{"LAZYREVIEW_HOME": ""})
	s.command.home = ""
	tr := i18n.EnglishTranslationSet()
	for _, argv := range [][]string{{"list"}, {"terminal"}, {"load", "https://github.com/octo/widgets/pull/30"}} {
		if err := s.command.Run(argv); err == nil || err.Error() != tr.NoDataHome {
			t.Errorf("%v with no home folder to start from says %v", argv, err)
		}
	}
	if len(s.loads)+len(s.opened) != 0 {
		t.Fatal("something loaded or opened with no data home")
	}
	if _, err := os.Stat(filepath.Join(".lazyreview")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a data home was made in the working folder")
	}
}
