package server

// PullRepo is one repository's open pull requests that involve an account,
// read as Account, in units by their lowest number.
type PullRepo struct {
	Host    string `json:"host"`
	Name    string `json:"name"`
	Account string `json:"account"`
	Units   []Unit `json:"units"`
}
