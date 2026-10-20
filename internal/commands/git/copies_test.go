package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

func cloned(t *testing.T, repo *testsupport.ThrowawayRepo, home string) string {
	t.Helper()
	bare := filepath.Join(home, "clones", "widgets.git")
	if out, err := exec.Command("git", "clone", "--quiet", "--bare", repo.Work, bare).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	return bare
}

func read(t *testing.T, path string) string {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

func TestACopyIsAtTheCommitOnTheBranchGivenWithNoRemoteBorrowingTheClonesObjects(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	repo := testsupport.NewThrowawayRepo(t)
	home := t.TempDir()
	base := repo.Commit("main", map[string]string{"src/money.ts": "export const a = 1;\n", "test/split.test.ts": "splitPence(1000, 3);\n"}, "base")
	head := repo.Commit("feature/split", map[string]string{"src/money.ts": "export const a = 2;\n"}, "split")
	repo.Checkout("main")
	source := cloned(t, repo, home)
	before := testsupport.FolderState(t, source)
	folder := filepath.Join(home, "copies", "github.com", "octo", "widgets", "7", "1")
	made, err := NewCopies(tr, nil, nil).Prepare(context.Background(), folder, CopyOf{Source: source, Commit: head, Branches: map[string]string{"main": base, "pull/7": head}, Branch: "pull/7"})
	if err != nil || made != folder {
		t.Fatalf("the copy is %q, %v", made, err)
	}
	if read(t, filepath.Join(folder, "src/money.ts")) != "export const a = 2;\n" || read(t, filepath.Join(folder, "test/split.test.ts")) != "splitPence(1000, 3);\n" {
		t.Error("the copy holds other code")
	}
	git := func(args ...string) string { return testsupport.RunGit(t, folder, args...) }
	if git("branch", "--show-current") != "pull/7" || git("rev-parse", "main") != base || git("remote") != "" || git("status", "--porcelain") != "" {
		t.Error("the copy's branches, remotes or state are wrong")
	}
	if !strings.Contains(read(t, filepath.Join(folder, ".git/objects/info/alternates")), "widgets.git/objects") {
		t.Error("the copy does not borrow the clone's objects")
	}
	if !reflect.DeepEqual(testsupport.FolderState(t, source), before) {
		t.Error("the clone it came from changed")
	}
	if left, _ := os.ReadDir(filepath.Dir(folder)); len(left) != 1 {
		t.Errorf("beside the copy are %v", left)
	}
}

func TestACopyIsMadeOnceThenOnlyMarkedUsedAndACommitTheCloneLacksSaysSo(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	repo := testsupport.NewThrowawayRepo(t)
	home := t.TempDir()
	commit := repo.Commit("main", map[string]string{"a.txt": "a\n"}, "base")
	source := cloned(t, repo, home)
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	copies := NewCopies(tr, nil, func() time.Time { return now })
	folder := filepath.Join(home, "copies", "1")
	of := CopyOf{Source: source, Commit: commit, Branches: map[string]string{"main": commit}, Branch: "main"}
	if _, err := copies.Prepare(context.Background(), folder, of); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "a.txt"), []byte("changed by Claude\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if _, err := copies.Prepare(context.Background(), folder, of); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(folder)
	if read(t, filepath.Join(folder, "a.txt")) != "changed by Claude\n" || !info.ModTime().Equal(now) {
		t.Errorf("the copy was made again, or not marked used at %s: %s", now, info.ModTime())
	}
	missing := strings.Repeat("0", 40)
	_, err := copies.Prepare(context.Background(), filepath.Join(home, "copies", "2"), CopyOf{Source: source, Commit: missing, Branches: map[string]string{"main": missing}, Branch: "main"})
	if err == nil || err.Error() != "commit 0000000 is no longer in "+source+"; load the review again" {
		t.Errorf("a missing commit gave %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "copies", "2")); err == nil {
		t.Error("a copy was left for a missing commit")
	}
}

func TestPruningRemovesEachCopyNotUsedNeverTheOneKeptAndTheFoldersLeftEmpty(t *testing.T) {
	home := t.TempDir()
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	aged := func(path string, age time.Duration) {
		if err := os.MkdirAll(filepath.Join(home, path, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(filepath.Join(home, path), now.Add(-age), now.Add(-age)); err != nil {
			t.Fatal(err)
		}
	}
	aged("github.com/octo/widgets/7/1", 2*day)
	aged("github.com/octo/widgets/7/2", time.Minute)
	aged("local/work-1a2b3c4d/feature~x/1", 3*day)
	aged("github.com/octo/widgets/9/1", 3*day)
	NewCopies(i18n.EnglishTranslationSet(), nil, func() time.Time { return now }).Prune(home, day, filepath.Join(home, "github.com/octo/widgets/9/1"))
	names := func(path string) []string {
		entries, _ := os.ReadDir(filepath.Join(home, path))
		var out []string
		for _, e := range entries {
			out = append(out, e.Name())
		}
		return out
	}
	if got := names("github.com/octo/widgets/7"); !reflect.DeepEqual(got, []string{"2"}) {
		t.Errorf("of #7 are left %v", got)
	}
	if got := names("github.com/octo/widgets/9"); !reflect.DeepEqual(got, []string{"1"}) {
		t.Errorf("of #9 are left %v", got)
	}
	if _, err := os.Stat(filepath.Join(home, "local")); err == nil {
		t.Error("a folder left empty stayed")
	}
}

func TestACloneOfARepositoryOnThisMachineFetchesOnlyTheCommitsAskedFor(t *testing.T) {
	repo := testsupport.NewThrowawayRepo(t)
	home := t.TempDir()
	base := repo.Commit("main", map[string]string{"a.txt": "a\n"}, "base")
	head := repo.Commit("feature/x", map[string]string{"a.txt": "b\n"}, "x")
	before := testsupport.FolderState(t, repo.Work)
	clone := filepath.Join(home, "clones", "local", "work-1a2b3c4d.git")
	got, err := NewClones(i18n.EnglishTranslationSet(), nil).CopyFrom(context.Background(), clone, repo.Work, []string{head, base})
	if err != nil || got != clone {
		t.Fatalf("the clone is %q, %v", got, err)
	}
	if testsupport.RunGit(t, clone, "rev-parse", "refs/commits/"+head) != head || testsupport.RunGit(t, clone, "rev-parse", "refs/commits/"+base) != base {
		t.Error("a commit is not kept under a ref of its own")
	}
	if !reflect.DeepEqual(testsupport.FolderState(t, repo.Work), before) {
		t.Error("the repository changed")
	}
}
