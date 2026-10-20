package models

// GitHubFile is one file of a pull request's diff as GitHub shows it, with
// its patch, which a binary or very large file has none of.
type GitHubFile struct {
	Filename string  `json:"filename"`
	Patch    *string `json:"patch"`
}
