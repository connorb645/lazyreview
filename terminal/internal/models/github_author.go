package models

// GitHubAuthor is who wrote a comment, as GitHub's GraphQL API gives it:
// their login and their kind, such as User or Bot.
type GitHubAuthor struct {
	Login    string `json:"login"`
	Typename string `json:"__typename"`
}
