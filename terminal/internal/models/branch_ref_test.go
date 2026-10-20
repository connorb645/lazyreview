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
	of, err := BranchRefOf(tr, "/work/widgets/.git", "feature/search")
	if err != nil || !strings.HasPrefix(of.ID(), "local/widgets-") || !strings.HasSuffix(of.ID(), "/feature~search") {
		t.Fatalf("a repository's branch is %q, %v", of.ID(), err)
	}
	if again, _ := BranchRefOf(tr, "/work/widgets/.git", "feature/search"); again != of {
		t.Fatal("the same repository and branch gave another id")
	}
	if elsewhere, _ := BranchRefOf(tr, "/elsewhere/widgets/.git", "feature/search"); elsewhere.ID() == of.ID() {
		t.Fatal("a repository of the same name elsewhere is the same repository")
	}
}

func TestARepositoryIsNamedForItsFolderOrABareOneForItselfAsANameAnAddressCanHold(t *testing.T) {
	for folder, name := range map[string]string{"/work/widgets/.git": "widgets", "/remotes/widgets.git": "widgets", "/work/.dot files/.git": "dot-files"} {
		if got := RepositoryName(folder); got != name {
			t.Errorf("%s is named %q, want %q", folder, got, name)
		}
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
