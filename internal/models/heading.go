package models

// Heading names the change: its title, and a line saying what it is, in
// parts, the bold ones names and commits.
type Heading struct {
	Title   string        `json:"title"`
	Context []HeadingPart `json:"context"`
	Head    string        `json:"head"`
	Base    string        `json:"base"`
}
