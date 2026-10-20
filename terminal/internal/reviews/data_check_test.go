package reviews

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/store"
)

// Every review of the data home LAZYREVIEW_DATA_CHECK names opens: its
// state, its page and each file it draws. It only reads, and says only how
// many reviews, comments, conversations and files it opened, and which
// part of which review, by its place in the list, it could not. With no
// data home named it is skipped.
func TestEveryReviewOfADataHomeOpens(t *testing.T) {
	root := os.Getenv("LAZYREVIEW_DATA_CHECK")
	if root == "" {
		t.Skip("LAZYREVIEW_DATA_CHECK names no data home")
	}
	tr := i18n.EnglishTranslationSet()
	reviews := store.NewReviewStore(tr, store.DataHome{Root: root}, store.NewLocks(tr))
	r := New(tr, reviews, nil, nil)
	ids := reviewIDs(t, filepath.Join(root, "reviews"))
	opened, pulls, branches, moved, comments, conversations, files := 0, 0, 0, 0, 0, 0, 0
	var failed []string
	for n, id := range ids {
		state, err := r.State(id)
		if err != nil {
			failed = append(failed, failure(n, "state", err))
			continue
		}
		if state.Kind == "branch" {
			branches++
		} else {
			pulls++
		}
		if state.Moved != nil {
			moved++
		}
		if state.Queue != nil {
			for _, m := range append(append([]models.QueueMember{}, state.Queue.Members...), state.Queue.Earlier...) {
				comments += len(m.Comments)
			}
		}
		conversations += len(state.Threads)
		page, err := r.Page(id)
		if err != nil {
			failed = append(failed, failure(n, "page", err))
			continue
		}
		bad := false
		for _, f := range append(append([]models.FileEntry{}, page.Model.Files...), page.Model.ContextFiles...) {
			if _, err := r.File(id, f.Path, page.Build); err != nil {
				failed = append(failed, failure(n, "file", err))
				bad = true
				break
			}
			files++
		}
		if !bad {
			opened++
		}
	}
	t.Logf("%d reviews found, %d opened whole: %d of pull requests, of them %d moved into a stack, and %d of branches", len(ids), opened, pulls, moved, branches)
	t.Logf("queued comments: %d, conversations with Claude: %d, files opened: %d", comments, conversations, files)
	for _, f := range failed {
		t.Errorf("%s", f)
	}
}

// reviewIDs is the id of every review folder under reviews, by its place on
// disk: <host>/<owner>/<repo>/<pull>, or local/<repository key>/<branch>.
func reviewIDs(t *testing.T, reviews string) []string {
	t.Helper()
	var ids []string
	err := filepath.WalkDir(reviews, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "review.json" {
			return nil
		}
		rel, err := filepath.Rel(reviews, filepath.Dir(path))
		if err != nil {
			return err
		}
		ids = append(ids, filepath.ToSlash(rel))
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("could not list the reviews: %v", err)
	}
	return ids
}

// failure names which part of the nth review would not open, and how, with
// no word of the review itself: no path, name or text of it.
func failure(n int, part string, err error) string {
	kind := "a refusal"
	switch {
	case strings.Contains(err.Error(), "could not read"):
		kind = "a file that could not be read"
	case strings.Contains(err.Error(), "not a review id"):
		kind = "a folder that names no review"
	}
	return "review " + strconv.Itoa(n+1) + ": its " + part + " did not open: " + kind
}
