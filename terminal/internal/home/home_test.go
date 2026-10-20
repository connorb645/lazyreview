package home

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/commands/github"
	"github.com/connorb645/lazyreview/terminal/internal/config"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/testsupport"
)

func configIn(t *testing.T, text string) (string, func() (config.Read, error)) {
	t.Helper()
	tr := i18n.EnglishTranslationSet()
	path := filepath.Join(t.TempDir(), "config.yml")
	if text != "" {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return path, func() (config.Read, error) {
		read, _ := os.ReadFile(path)
		return config.ConfigFrom(tr, string(read), path, "/home/reviewer")
	}
}

func TestASettingsFileAnEarlierVersionLeftSaysHowToMoveIt(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "settings.json"), []byte(`{"accounts": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	path, read := configIn(t, "")
	reviews := store.NewReviewStore(tr, store.DataHome{Root: root}, store.NewLocks(tr))
	gh := &testsupport.FakeGh{}
	_, err := NewOpenPulls(tr, reviews, read, func(host, account string) *github.GitHub { return github.New(tr, host, account, gh.Run) }).List()
	if err == nil || err.Error() != filepath.Join(root, "settings.json")+" is no longer read; lazyreview config move moves its accounts into "+path {
		t.Errorf("a settings file left gave %v", err)
	}
}

func TestAnEntryThatIsNotARepositoryIsRefusedAndNothingIsWritten(t *testing.T) {
	path, read := configIn(t, "github:\n  accounts: []\n")
	_, err := NewHiddenRepositories(i18n.EnglishTranslationSet(), read).Change("octo", nil, []string{"octo/widgets"})
	if err == nil || err.Error() != `"octo" is not a repository as owner/name, or owner/* for every repository of the owner` {
		t.Errorf("an owner alone gave %v", err)
	}
	if text, _ := os.ReadFile(path); string(text) != "github:\n  accounts: []\n" {
		t.Errorf("the config now reads %q", text)
	}
}

func TestChangesAskedAtOnceAreMadeOneAtATimeSoNoneIsLost(t *testing.T) {
	path, read := configIn(t, "")
	hidden := NewHiddenRepositories(i18n.EnglishTranslationSet(), read)
	var wait sync.WaitGroup
	for _, name := range []string{"a/b", "c/d", "e/*", "f/g", "h/i"} {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := hidden.Change(name, nil, nil); err != nil {
				t.Error(err)
			}
		}()
	}
	wait.Wait()
	text, _ := os.ReadFile(path)
	for _, name := range []string{"a/b", "c/d", "e/*", "f/g", "h/i"} {
		if !strings.Contains(string(text), "    - "+name+"\n") {
			t.Errorf("%s was lost from\n%s", name, text)
		}
	}
}
