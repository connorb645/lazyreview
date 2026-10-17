package types

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// FilesLoaded is a component's files loaded to show its diff at a place, or
// why they could not be.
type FilesLoaded struct {
	Component string
	Files     map[string]server.File
	Place     *server.Lines
	Err       error
}
