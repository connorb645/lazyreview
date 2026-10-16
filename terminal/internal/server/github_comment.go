package server

// GitHubComment is one comment of a review thread on GitHub.
type GitHubComment struct {
	Author string `json:"author"`
	Bot    bool   `json:"bot"`
	Body   string `json:"body"`
	At     string `json:"at"`
	URL    string `json:"url"`
}
