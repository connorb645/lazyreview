package models

import (
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
)

func TestAPullRequestIsReadFromItsAddressItsReviewIDOrItsShortForm(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for _, text := range []string{"https://github.com/octo/widgets/pull/12", "https://github.com/octo/widgets/pull/12/files#diff-1", "github.com/octo/widgets/12", "octo/widgets#12"} {
		ref, err := ParsePullRef(tr, text)
		if err != nil || ref != (PullRef{Host: "github.com", Repo: "octo/widgets", Pull: 12}) || ref.ID() != "github.com/octo/widgets/12" || ref.URL() != "https://github.com/octo/widgets/pull/12" {
			t.Errorf("%s reads as %+v, %v", text, ref, err)
		}
	}
	if ref, _ := ParsePullRef(tr, "https://git.example.com/team/app/pull/3"); ref.ID() != "git.example.com/team/app/3" {
		t.Errorf("another host reads as %s", ref.ID())
	}
	if ref, _ := ParsePullRef(tr, "octo/widgets#12"); ref.WithPull(13).ID() != "github.com/octo/widgets/13" {
		t.Errorf("the same repository's other pull request is %s", ref.WithPull(13).ID())
	}
}

func TestAnythingElseIsRefusedIncludingNamesThatWouldClimbOutOfAFolder(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for _, text := range []string{"", "https://github.com/octo/widgets/issues/12", "octo#12", "https://github.com/../x/pull/1", "github.com/octo/../1", "octo/widgets#0"} {
		if _, err := ParsePullRef(tr, text); err == nil || !strings.Contains(err.Error(), "pull request") {
			t.Errorf("%q is taken, or refused as %v", text, err)
		}
	}
}

func TestAReviewIDNamesAPullRequestsReviewOrABranchsAndNothingElse(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	if ref, err := ParseReviewRef(tr, "github.com/octo/widgets/12"); err != nil || ref.ID() != "github.com/octo/widgets/12" {
		t.Errorf("a pull request's review id reads as %v, %v", ref, err)
	}
	if ref, err := ParseReviewRef(tr, "local/widgets-1a2b3c4d/fix%25wording"); err != nil || ref.(BranchRef).Branch != "fix%wording" {
		t.Errorf("a branch review's id, written as an address, reads as %v, %v", ref, err)
	}
	for _, id := range []string{"github.com/octo/widgets", "github.com/octo/widgets/x", "local/widgets-1a2b3c4d", "%zz/a/b/1"} {
		if _, err := ParseReviewRef(tr, id); err == nil {
			t.Errorf("%q is taken as a review id", id)
		}
	}
}
