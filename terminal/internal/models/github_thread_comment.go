package models

// GitHubThreadComment is one comment of a review thread on GitHub, as the
// last load kept it: its id, its author and whether that is a bot, its
// text, when it was made and its address.
type GitHubThreadComment struct {
	ID     int    `json:"id"`
	Author string `json:"author"`
	Bot    bool   `json:"bot"`
	Body   string `json:"body"`
	At     string `json:"at"`
	URL    string `json:"url"`
}
