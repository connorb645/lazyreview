package queue

import "github.com/connorb645/lazyreview/terminal/internal/commands/github"

// GitHubFor is the GitHub a review talks to: its host, as its account.
type GitHubFor func(host, account string) *github.GitHub
