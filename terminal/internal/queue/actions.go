// Package queue keeps a review's queue: the comments waiting to go on each
// pull request, where a comment on lines goes, and the three actions that
// send a review.
package queue

// Action is one of the three ways a review is sent: the event GitHub takes,
// the state GitHub then gives the review, and whether it sends the queued
// comments.
type Action struct {
	Event    string
	State    string
	Comments bool
}

// Actions is every action, by name, in the order they are offered. No action
// requests changes.
var Actions = []string{"comment", "approve-with-comments", "approve"}

// ActionOf is each action by name.
var ActionOf = map[string]Action{
	"comment":               {Event: "COMMENT", State: "COMMENTED", Comments: true},
	"approve-with-comments": {Event: "APPROVE", State: "APPROVED", Comments: true},
	"approve":               {Event: "APPROVE", State: "APPROVED", Comments: false},
}
