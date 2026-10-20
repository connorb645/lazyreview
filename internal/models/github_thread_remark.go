package models

// GitHubThreadRemark is one comment of a review thread as GitHub's GraphQL
// API gives it, with the end of the diff it was made on. Author is nil for
// an account that no longer exists.
type GitHubThreadRemark struct {
	DatabaseID int           `json:"databaseId"`
	Author     *GitHubAuthor `json:"author"`
	Body       string        `json:"body"`
	CreatedAt  string        `json:"createdAt"`
	URL        string        `json:"url"`
	DiffHunk   string        `json:"diffHunk"`
}
