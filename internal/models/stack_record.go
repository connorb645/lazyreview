package models

// StackRecord is a stack's part of its page: each member's head, start and
// counts, where a comment on each line of the combined diff goes, by path
// and side, and the warnings about a stack out of step.
type StackRecord struct {
	Members  []StackMember                 `json:"members"`
	Lines    map[string]map[string]LineMap `json:"lines"`
	Warnings []string                      `json:"warnings"`
}
