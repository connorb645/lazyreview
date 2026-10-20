// Package page builds what the terminal app draws for one review: the
// diagram's model and layout, the diff split per file, the whole text of
// changed files, the code of unchanged context, and for a stack, where a
// comment on each line goes.
package page

import (
	"strconv"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

// DescribeChange is the line under the page's title that says what the
// change is: the subject, then where it sits, such as Pull request **#12**
// on **main**: head **3f2a9c1d**, merge base **77b0e412**, 3 commits.
func DescribeChange(tr *i18n.TranslationSet, subject []models.HeadingPart, base, headCommit, mergeBase string, commits int) []models.HeadingPart {
	parts := append([]models.HeadingPart{}, subject...)
	if base != "" {
		parts = append(parts, models.HeadingPart{Text: tr.ChangeOn}, models.HeadingPart{Text: base, Bold: true})
	}
	return append(parts,
		models.HeadingPart{Text: tr.ChangeHead}, models.HeadingPart{Text: utils.Short(headCommit), Bold: true},
		models.HeadingPart{Text: tr.ChangeMergeBase}, models.HeadingPart{Text: utils.Short(mergeBase), Bold: true},
		models.HeadingPart{Text: utils.ResolvePlaceholderString(tr.ChangeCommits, map[string]string{"commits": utils.Plural(commits, tr.CommitNoun)})})
}

// PullsSubject is the subject of a pull request, or of a stack of them,
// bottom first, naming its bottom and top.
func PullsSubject(tr *i18n.TranslationSet, pulls []int) []models.HeadingPart {
	if len(pulls) == 1 {
		return []models.HeadingPart{{Text: tr.SubjectPull}, {Text: "#" + strconv.Itoa(pulls[0]), Bold: true}}
	}
	return []models.HeadingPart{
		{Text: tr.SubjectStack}, {Text: "#" + strconv.Itoa(pulls[0]), Bold: true},
		{Text: tr.SubjectTo}, {Text: "#" + strconv.Itoa(pulls[len(pulls)-1]), Bold: true},
	}
}

// BranchSubject is the subject of a branch with no pull request.
func BranchSubject(tr *i18n.TranslationSet, branch string) []models.HeadingPart {
	return []models.HeadingPart{{Text: tr.SubjectBranch}, {Text: branch, Bold: true}}
}
