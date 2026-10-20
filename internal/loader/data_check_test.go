package loader

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/build/components"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/store"
)

// Everything a load reads back from the data home LAZYREVIEW_DATA_CHECK
// names reads: each repository's reviews, as a load finds where a stack's
// review lives, and each review's components config. It only reads, and
// says only how many it read, and which, by its place in the list, it
// could not. With no data home named it is skipped.
func TestEveryReviewOfADataHomeReadsAsALoadReadsIt(t *testing.T) {
	root := os.Getenv("LAZYREVIEW_DATA_CHECK")
	if root == "" {
		t.Skip("LAZYREVIEW_DATA_CHECK names no data home")
	}
	tr := i18n.EnglishTranslationSet()
	reviews := store.NewReviewStore(tr, store.DataHome{Root: root}, store.NewLocks(tr))
	repositories, listed, configs := 0, 0, 0
	var failed []string
	hosts, _ := os.ReadDir(filepath.Join(root, "reviews"))
	for _, host := range hosts {
		if !host.IsDir() || host.Name() == "local" {
			continue
		}
		owners, _ := os.ReadDir(filepath.Join(root, "reviews", host.Name()))
		for _, owner := range owners {
			names, _ := os.ReadDir(filepath.Join(root, "reviews", host.Name(), owner.Name()))
			for _, name := range names {
				repositories++
				held, err := reviews.InRepository(host.Name(), owner.Name()+"/"+name.Name())
				if err != nil {
					failed = append(failed, "repository "+strconv.Itoa(repositories)+": its reviews could not be listed")
					continue
				}
				listed += len(held)
			}
		}
	}
	var ids []string
	_ = filepath.WalkDir(filepath.Join(root, "reviews"), func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "review.json" {
			rel, _ := filepath.Rel(filepath.Join(root, "reviews"), filepath.Dir(path))
			ids = append(ids, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(ids)
	for n, id := range ids {
		ref, err := models.ParseReviewRef(tr, id)
		if err != nil {
			failed = append(failed, "review "+strconv.Itoa(n+1)+": its folder names no review")
			continue
		}
		raw, err := reviews.ReadConfig(ref)
		if err != nil {
			failed = append(failed, "review "+strconv.Itoa(n+1)+": its config could not be read")
			continue
		}
		if raw == nil {
			continue
		}
		if _, err := components.Read(tr, raw); err != nil {
			failed = append(failed, "review "+strconv.Itoa(n+1)+": its config did not read: "+strconv.Itoa(strings.Count(err.Error(), "\n"))+" problems")
			continue
		}
		configs++
	}
	t.Logf("%d repositories, %d pull request reviews listed by repository, %d reviews in all, %d components configs read", repositories, listed, len(ids), configs)
	for _, f := range failed {
		t.Errorf("%s", f)
	}
}
