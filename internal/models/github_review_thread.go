package models

// GitHubReviewThread is a review thread on a pull request as GitHub's
// GraphQL API gives it: where it is, whether it is resolved or outdated,
// who resolved it, and its first comments with their count.
type GitHubReviewThread struct {
	ID                string               `json:"id"`
	IsResolved        bool                 `json:"isResolved"`
	IsOutdated        bool                 `json:"isOutdated"`
	Path              string               `json:"path"`
	Line              *int                 `json:"line"`
	StartLine         *int                 `json:"startLine"`
	OriginalLine      *int                 `json:"originalLine"`
	OriginalStartLine *int                 `json:"originalStartLine"`
	DiffSide          string               `json:"diffSide"`
	StartDiffSide     *string              `json:"startDiffSide"`
	SubjectType       string               `json:"subjectType"`
	ResolvedBy        *GitHubUser          `json:"resolvedBy"`
	Comments          GitHubThreadComments `json:"comments"`
}
