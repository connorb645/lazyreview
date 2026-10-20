package models

// GitHubThreadComments is the comments of a review thread as GitHub's
// GraphQL API gives them: the first of them, and how many it holds.
type GitHubThreadComments struct {
	TotalCount int                  `json:"totalCount"`
	Nodes      []GitHubThreadRemark `json:"nodes"`
}
