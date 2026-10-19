package types

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// HiddenChanged is the config's hidden repositories after a change asked
// on the home screen: the entry hidden, or the entries taken off, and what
// now hides each repository, or why the change failed.
type HiddenChanged struct {
	Hide   string
	Show   []string
	Hidden server.Hidden
	Err    error
}
