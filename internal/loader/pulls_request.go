package loader

// PullsRequest is a load of pull requests: their addresses, of one
// repository, bottom first, two or more being a stack, each based on the
// head branch of the one before. Config is a components config as JSON,
// Context whether to draw the unchanged code the change touches, and
// Account the GitHub account to act as; each left nil keeps what the
// review had, or the default.
type PullsRequest struct {
	Links   []string
	Config  []byte
	Context *bool
	Account *string
}
