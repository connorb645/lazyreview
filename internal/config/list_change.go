package config

// ListChange is a change to a list setting: the items to put at its end,
// and the items to take out.
type ListChange struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}
