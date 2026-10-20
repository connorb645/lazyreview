package config

// Account is a GitHub account the home screen lists the open pull requests
// of, and the teams a review asked of counts as asked of it.
type Account struct {
	Login string `json:"login"`
	Teams []Team `json:"teams"`
}
