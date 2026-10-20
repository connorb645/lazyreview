package config

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

const (
	hiddenKey = "github.hiddenRepositories"
	written   = "# Mine.\ngithub:\n  accounts:\n    # - login: old\n    - login: lee\n"
)

func names(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func TestAChangeIsSavedInPlaceKeepingTheFilesModeWithNoOtherFileLeft(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(file, []byte(written), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SaveListChange(i18n.EnglishTranslationSet(), file, hiddenKey, ListChange{Add: []string{"octo/*"}}); err != nil {
		t.Fatal(err)
	}
	text, _ := os.ReadFile(file)
	info, _ := os.Stat(file)
	if string(text) != written+"  hiddenRepositories:\n    - octo/*\n" || info.Mode().Perm() != 0o600 || !reflect.DeepEqual(names(t, dir), []string{"config.yml"}) {
		t.Errorf("the file reads %q, mode %v, beside %v", text, info.Mode().Perm(), names(t, dir))
	}
}

func TestAConfigThatIsALinkKeepsItsLinkAndTheFileItPointsToChanges(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles", "lazyreview.yml")
	link := filepath.Join(dir, "config.yml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(written), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := SaveListChange(i18n.EnglishTranslationSet(), link, hiddenKey, ListChange{Add: []string{"octo/widgets"}}); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Lstat(link)
	text, _ := os.ReadFile(target)
	if info.Mode()&os.ModeSymlink == 0 || !strings.HasSuffix(string(text), "hiddenRepositories:\n    - octo/widgets\n") {
		t.Errorf("the link is %v, and the file reads %q", info.Mode(), text)
	}
}

func TestWithNoFileOneIsMadeAndItsFolderHoldingOnlyTheList(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	dir := t.TempDir()
	file := filepath.Join(dir, "lazyreview", "config.yml")
	if err := SaveListChange(tr, file, hiddenKey, ListChange{Add: []string{"octo/*"}}); err != nil {
		t.Fatal(err)
	}
	text, _ := os.ReadFile(file)
	if string(text) != "github:\n  hiddenRepositories:\n    - octo/*\n" {
		t.Errorf("the new file reads %q", text)
	}
	if err := SaveListChange(tr, filepath.Join(dir, "other", "config.yml"), hiddenKey, ListChange{Remove: []string{"octo/*"}}); err != nil || !reflect.DeepEqual(names(t, dir), []string{"lazyreview"}) {
		t.Errorf("taking off what no file holds made %v, %v", names(t, dir), err)
	}
}

func TestASaveThatFailsOrIsRefusedLeavesTheOldFileWholeAndSaysWhy(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	dir := t.TempDir()
	file := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(file, []byte(written), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	err := SaveListChange(tr, file, hiddenKey, ListChange{Add: []string{"octo/*"}})
	_ = os.Chmod(dir, 0o755)
	if err == nil || !regexp.MustCompile(`^`+regexp.QuoteMeta(file)+` was not changed: .*permission denied.*; change it by hand, as docs/Config.md shows$`).MatchString(err.Error()) {
		t.Errorf("a folder it cannot write gave %v", err)
	}
	if text, _ := os.ReadFile(file); string(text) != written || !reflect.DeepEqual(names(t, dir), []string{"config.yml"}) {
		t.Errorf("the old file reads %q beside %v", text, names(t, dir))
	}
	if err := os.WriteFile(file, []byte("github: {accounts: []}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SaveListChange(tr, file, hiddenKey, ListChange{Add: []string{"octo/*"}}); err == nil || !strings.Contains(err.Error(), "was not changed: github is written in a way lazyreview cannot change line by line") {
		t.Errorf("a refused change gave %v", err)
	}
}
