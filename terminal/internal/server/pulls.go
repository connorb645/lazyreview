package server

// Pulls is what the home screen lists: the open pull requests that involve
// the accounts in the settings file, by repository, and how reading each
// account went. Settings is the settings file's path; with no accounts in
// it, there is nothing to list.
type Pulls struct {
	Settings string        `json:"settings"`
	Accounts []AccountRead `json:"accounts"`
	Repos    []PullRepo    `json:"repos"`
}
