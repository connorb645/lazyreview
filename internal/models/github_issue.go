package models

// GitHubIssue is one issue or pull request a search found: its number and
// the address of its repository.
type GitHubIssue struct {
	Number        int    `json:"number"`
	RepositoryURL string `json:"repository_url"`
}
