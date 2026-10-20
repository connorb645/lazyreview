package config

// Team is a team of an account, organisation/team, and the date from which
// a pull request opened counts, or none.
type Team struct {
	Name  string  `json:"name"`
	Since *string `json:"since"`
}
