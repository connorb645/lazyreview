package stack

import "github.com/connorb645/lazyreview/terminal/internal/models"

// Mapped is where a comment on each line of a stack's combined diff goes,
// by path, then side, RIGHT for the top head and LEFT for the combined base,
// then line; and which pull requests changed each file, by path.
type Mapped struct {
	Lines map[string]map[string]models.LineMap
	Files map[string]FilePulls
}
