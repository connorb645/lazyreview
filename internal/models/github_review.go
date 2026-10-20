package models

// GitHubReview is a review on a pull request as GitHub gives it: who sent
// it, at which commit, how it stands and its address.
type GitHubReview struct {
	ID       int        `json:"id"`
	User     GitHubUser `json:"user"`
	CommitID string     `json:"commit_id"`
	State    string     `json:"state"`
	HTMLURL  string     `json:"html_url"`
}
