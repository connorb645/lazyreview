package models

// GitHubReviewComment is a comment of a review on a pull request's lines, as
// GitHub gives it: the review it belongs to, its path and its text.
type GitHubReviewComment struct {
	ReviewID int    `json:"pull_request_review_id"`
	Path     string `json:"path"`
	Body     string `json:"body"`
}
