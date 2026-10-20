package models

import (
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

func TestABranchReviewIDNamesTheRepositoryAndTheBranchAndReadsBack(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	ref, err := NewBranchRef(tr, "widgets-1a2b3c4d", "feature/search")
	if err != nil || ref.ID() != "local/widgets-1a2b3c4d/feature~search" {
		t.Fatalf("the id is %q, %v", ref.ID(), err)
	}
	back, err := ParseBranchRef(tr, ref.ID())
	if err != nil || back != ref {
		t.Fatalf("the id reads back as %+v, %v", back, err)
	}
}

func TestOnlyABranchNameAndAKeyAReviewFolderCanSafelyHoldAreTaken(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for _, id := range []string{"local/widgets-1a2b3c4d/..", "local/../feature", "local/widgets-1a2b3c4d/.hidden", "local/widgets/feature",
		"local/widgets-1a2b3c4d/a~~b", "local/widgets-1a2b3c4d/a b", "github.com/octo/widgets/1", "local/widgets-1a2b3c4d/x.lock"} {
		if _, err := ParseBranchRef(tr, id); err == nil || !strings.Contains(err.Error(), "not a branch review") {
			t.Errorf("%q is taken, or refused as %v", id, err)
		}
	}
	if ref, err := ParseBranchRef(tr, "local/widgets-1a2b3c4d/fix#2"); err != nil || ref.Branch != "fix#2" {
		t.Errorf("fix#2 reads as %+v, %v", ref, err)
	}
}
