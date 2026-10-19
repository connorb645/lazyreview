package models

// PullRepo is one repository's open pull requests that involve an account,
// read as Account, in units by their lowest number. HiddenBy holds the
// entries of the config's hidden repositories that hide it from the home
// screen, or none.
type PullRepo struct {
	Host     string   `json:"host"`
	Name     string   `json:"name"`
	Account  string   `json:"account"`
	HiddenBy []string `json:"hidden_by"`
	Units    []Unit   `json:"units"`
}
