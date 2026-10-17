package config

// KeybindingPostingConfig is posting's keys. Down, Up, Choose, Open and
// Close are those of the choice of an action, Choose holding the keys of the
// choices 1 to 3, each by its place; Post and Back are those of what it
// sends, shown before it is sent.
type KeybindingPostingConfig struct {
	Down   []string `json:"down"`
	Up     []string `json:"up"`
	Choose []string `json:"choose"`
	Open   []string `json:"open"`
	Close  []string `json:"close"`
	Post   []string `json:"post"`
	Back   []string `json:"back"`
}
