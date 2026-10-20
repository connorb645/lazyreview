package diff

import (
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/commands/git"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

func numbered(prefix string, n int) string {
	var out strings.Builder
	for i := 1; i <= n; i++ {
		out.WriteString(prefix + " " + strings.Repeat("x", i%7) + "\n")
	}
	return out.String()
}

func fileNamed(t *testing.T, files []File, path string) File {
	t.Helper()
	for _, f := range files {
		if f.Path == path {
			return f
		}
	}
	t.Fatalf("no file %s", path)
	return File{}
}

func TestReadDiffReadsEachFileWithItsCountsTextsAndWhetherItStartsClosed(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	r := testsupport.NewThrowawayRepo(t)
	r.Commit("main", map[string]string{
		"src/Cache.swift":   "final class Cache {}\n",
		"src/Gone.swift":    "struct Gone {}\n",
		"docs/guide.md":     "# Guide\n",
		"data/rows.json":    "[]\n",
		"data/small.json":   "[]\n",
		"package-lock.json": "{}\n",
	}, "base")
	r.Git("checkout", "-q", "-b", "feature")
	r.Git("rm", "-q", "src/Gone.swift")
	r.Commit("feature", map[string]string{
		"src/Cache.swift":   "final class Cache {\n    let size = 1\n}\n",
		"src/New.swift":     "struct New {}\n",
		"docs/guide.md":     "# The guide\n",
		"data/rows.json":    numbered("row", 61),
		"data/small.json":   numbered("row", 3),
		"package-lock.json": "{\"a\": 1}\n",
		"big.txt":           numbered("line", 1501),
		"art/logo.svg":      "<svg/>\n",
	}, "change")
	read, err := ReadDiff(tr, git.NewRepository(tr, r.Work, nil), "main", "feature", false)
	if err != nil {
		t.Fatal(err)
	}
	if read.MergeBase != r.Git("rev-parse", "main") || read.HeadCommit != r.Git("rev-parse", "feature") || read.BaseCommit != read.MergeBase {
		t.Errorf("read between %s and %s, merge base %s", read.BaseCommit, read.HeadCommit, read.MergeBase)
	}
	if len(read.Files) != len(read.Chunks) || len(read.Files) != 9 {
		t.Fatalf("%d files from %d chunks", len(read.Files), len(read.Chunks))
	}
	cache := fileNamed(t, read.Files, "src/Cache.swift")
	if cache.Adds != 3 || cache.Dels != 1 || cache.Text == nil || cache.BaseText == nil || *cache.BaseText != "final class Cache {}\n" {
		t.Errorf("changed code keeps both texts and its counts: %+v", cache)
	}
	if gone := fileNamed(t, read.Files, "src/Gone.swift"); gone.Text != nil || gone.BaseText == nil || gone.Status != "D" {
		t.Errorf("a deleted file keeps only its base text: %+v", gone)
	}
	if guide := fileNamed(t, read.Files, "docs/guide.md"); guide.Text == nil || guide.BaseText != nil {
		t.Errorf("a changed file no language reads needs no base text: %+v", guide)
	}
	if added := fileNamed(t, read.Files, "src/New.swift"); added.BaseText != nil || added.OldPath != "" {
		t.Errorf("an added file has no base: %+v", added)
	}
	for path, closed := range map[string]bool{
		"data/rows.json": true, "data/small.json": false, "package-lock.json": true, "big.txt": true, "art/logo.svg": true, "src/Cache.swift": false,
	} {
		if got := fileNamed(t, read.Files, path).Collapsed; got != closed {
			t.Errorf("%s starts closed: %v", path, got)
		}
	}
}

func TestReadDiffReadsFromTheBaseItselfWhenExact(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	r := testsupport.NewThrowawayRepo(t)
	r.Commit("main", map[string]string{"a.txt": "a\n"}, "base")
	r.Commit("feature", map[string]string{"b.txt": "b\n"}, "feature")
	r.Checkout("main")
	r.Commit("main", map[string]string{"c.txt": "c\n"}, "main moves")
	repo := git.NewRepository(tr, r.Work, nil)
	merged, err := ReadDiff(tr, repo, "main", "feature", false)
	if err != nil {
		t.Fatal(err)
	}
	exact, err := ReadDiff(tr, repo, "main", "feature", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Files) != 1 || len(exact.Files) != 2 || exact.MergeBase != r.Git("rev-parse", "main") {
		t.Errorf("from the merge base %d files, from main itself %d", len(merged.Files), len(exact.Files))
	}
}

func TestReadDiffRefusesAChangeWithNoDifference(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	r := testsupport.NewThrowawayRepo(t)
	r.Commit("main", map[string]string{"a.txt": "a\n"}, "base")
	repo := git.NewRepository(tr, r.Work, nil)
	if _, err := ReadDiff(tr, repo, "main", "main", false); err == nil || err.Error() != "there is no difference between main and main" {
		t.Errorf("got %v", err)
	}
	if _, err := ReadDiff(tr, repo, "nowhere", "main", false); err == nil || err.Error() != `"nowhere" is not a commit in this repository` {
		t.Errorf("got %v", err)
	}
}
