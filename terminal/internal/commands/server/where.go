package server

// Where is where a comment on lines as the app shows them would go: on one
// pull request's own lines, or on one of several to choose from, with the
// server's sentence for it, or why none can.
type Where struct {
	Goes    *Target  `json:"goes"`
	Choose  []Choice `json:"choose"`
	Words   string   `json:"words"`
	Refused string   `json:"refused"`
}
