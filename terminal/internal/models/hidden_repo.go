package models

// HiddenRepo is a repository and the entries of the config's hidden
// repositories that hide it, or none.
type HiddenRepo struct {
	Name     string   `json:"name"`
	HiddenBy []string `json:"hidden_by"`
}
