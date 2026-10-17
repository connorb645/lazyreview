package server

// AccountRead is how reading one account's pull requests from GitHub went:
// Error says why it could not be read, or is empty.
type AccountRead struct {
	Login string `json:"login"`
	Error string `json:"error"`
}
