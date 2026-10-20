package models

// GitHubPull is a pull request as GitHub's API gives it: what lazyreview
// reads of it.
type GitHubPull struct {
	Number   int              `json:"number"`
	Title    string           `json:"title"`
	Body     string           `json:"body"`
	HTMLURL  string           `json:"html_url"`
	State    string           `json:"state"`
	MergedAt *string          `json:"merged_at"`
	ClosedAt *string          `json:"closed_at"`
	Draft    bool             `json:"draft"`
	User     GitHubUser       `json:"user"`
	Base     GitHubPullBranch `json:"base"`
	Head     GitHubPullBranch `json:"head"`
}
