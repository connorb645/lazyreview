package models

// GitHubPullBranch is a pull request's base or head as GitHub gives it: the
// branch, its commit and the repository it is in, which a deleted fork has
// none of.
type GitHubPullBranch struct {
	Ref  string      `json:"ref"`
	Sha  string      `json:"sha"`
	Repo *GitHubRepo `json:"repo"`
}
