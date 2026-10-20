package models

// GitHubSearch is one page of what a search found.
type GitHubSearch struct {
	Items []GitHubIssue `json:"items"`
}
