package store

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/connorb645/lazyreview/internal/models"
)

func TestTheStoreListsEveryReviewNewestFirstAndFindsTheOneHoldingAPullRequest(t *testing.T) {
	s, _ := scratch(t)
	a := models.PullRef{Host: "github.com", Repo: "octo/widgets", Pull: 1}
	b := models.PullRef{Host: "github.com", Repo: "octo/widgets", Pull: 2}
	branch := models.BranchRef{RepoKey: "widgets-1a2b3c4d", Branch: "feature/search"}
	writes := []struct {
		ref    models.ReviewRef
		review models.Review
	}{
		{a, models.Review{ID: a.ID(), LoadedAt: "2026-10-15T10:00:00Z", Members: []models.ReviewMember{{Pull: 1}, {Pull: 2}}}},
		{b, models.Review{ID: b.ID(), LoadedAt: "2026-10-15T09:00:00Z", Moved: &models.Moved{To: a.ID()}, Members: []models.ReviewMember{{Pull: 2}}}},
		{branch, models.Review{ID: branch.ID(), Kind: "branch", LoadedAt: "2026-10-15T11:00:00Z"}},
	}
	refs := []models.ReviewRef{a, b, branch}
	err := s.Change(refs, func() ([]FileWrite, error) {
		var files []FileWrite
		for _, w := range writes {
			f, err := s.RecordWrite(w.ref, w.review)
			if err != nil {
				return nil, err
			}
			files = append(files, f)
		}
		return files, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(s.Home().Repository("github.com", "octo/widgets"), "3"), 0o755); err != nil {
		t.Fatal(err)
	}
	kept, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, k := range kept {
		ids = append(ids, k.Ref.ID())
	}
	if want := []string{branch.ID(), a.ID(), b.ID()}; !slices.Equal(ids, want) {
		t.Fatalf("the store lists %v, want %v: newest load first, a folder with no record left out", ids, want)
	}
	held, rev, err := s.Find(b)
	if err != nil || rev == nil || held != a {
		t.Fatalf("#2 is found in %v, %v; want the stack's review %s", held, err, a.ID())
	}
	if _, rev, err := s.Find(models.PullRef{Host: "github.com", Repo: "octo/widgets", Pull: 3}); err != nil || rev != nil {
		t.Fatalf("a pull request no review holds is found as %+v, %v", rev, err)
	}
}

func TestAnEmptyDataHomeListsNoReview(t *testing.T) {
	s, _ := scratch(t)
	kept, err := s.List()
	if err != nil || len(kept) != 0 {
		t.Fatalf("an empty data home lists %v, %v", kept, err)
	}
}
