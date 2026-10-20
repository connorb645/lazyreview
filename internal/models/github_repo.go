package models

// GitHubRepo is a repository as GitHub gives it: its owner/name and the
// address a clone fetches from.
type GitHubRepo struct {
	FullName string `json:"full_name"`
	CloneURL string `json:"clone_url"`
}
