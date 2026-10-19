package types

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// FilesLoaded is files loaded to show their diff at a place, or why they
// could not be.
type FilesLoaded struct {
	Files map[string]server.File
	Place *server.Lines
	Err   error
}
