package models

// Pulls is what the home screen lists: the open pull requests that involve
// the accounts in the config, by repository, and how reading each account
// went. Config is the config file's path; with no accounts in it, there is
// nothing to list.
type Pulls struct {
	Config   string        `json:"config"`
	Accounts []AccountRead `json:"accounts"`
	Repos    []PullRepo    `json:"repos"`
}
