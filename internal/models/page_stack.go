package models

// PageStack is a stack's page data: each pull request's own diff, bottom
// first, and any warnings about the stack.
type PageStack struct {
	Members  []StackMember `json:"members"`
	Warnings []string      `json:"warnings"`
}
