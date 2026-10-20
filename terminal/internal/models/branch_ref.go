package models

import (
	"regexp"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

var (
	branchKey      = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9_.-]*-[0-9a-f]{8}$`)
	branchID       = regexp.MustCompile(`^local/([^/]+)/([^/]+)$`)
	branchRefusals = regexp.MustCompile(`[\x00-\x20~^:?*[\\\x7f]|\.\.|@\{|//|^/|/$|\.$`)
)

// BranchRef names one branch of a repository on this machine, for a review
// of a change that has no pull request yet. RepoKey is the repository's
// name with a short hash of its shared git folder, so two repositories of
// the same name never share a review.
type BranchRef struct {
	RepoKey string
	Branch  string
}

// NewBranchRef is branch of the repository repoKey names, or an error saying
// these name no branch review.
func NewBranchRef(tr *i18n.TranslationSet, repoKey, branch string) (BranchRef, error) {
	if !branchKey.MatchString(repoKey) || !isBranchName(branch) {
		return BranchRef{}, refError(tr.NotABranch, map[string]string{"repoKey": repoKey, "branch": branch})
	}
	return BranchRef{RepoKey: repoKey, Branch: branch}, nil
}

// ParseBranchRef reads a branch review's id back, such as
// local/widgets-1a2b3c4d/feature~search.
func ParseBranchRef(tr *i18n.TranslationSet, text string) (BranchRef, error) {
	m := branchID.FindStringSubmatch(text)
	if m == nil {
		return BranchRef{}, refError(tr.NotABranchID, map[string]string{"text": text})
	}
	return NewBranchRef(tr, m[1], strings.ReplaceAll(m[2], "~", "/"))
}

// ID is local/<repository key>/<branch>, each / of the branch written as ~,
// which no branch name holds: the review's folder under the data home and
// its address in the app.
func (r BranchRef) ID() string {
	return "local/" + r.RepoKey + "/" + strings.ReplaceAll(r.Branch, "/", "~")
}

func isBranchName(name string) bool {
	if name == "" || name == "@" || branchRefusals.MatchString(name) {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return false
		}
	}
	return true
}
