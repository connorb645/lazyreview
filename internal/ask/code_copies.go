package ask

import (
	"context"
	"time"

	"github.com/connorb645/lazyreview/internal/commands/git"
)

// CodeCopies makes the throwaway copies of the code Claude answers in,
// removes those no question used for a while, and removes one together
// with its conversation.
type CodeCopies interface {
	Prepare(ctx context.Context, folder string, of git.CopyOf) (string, error)
	Prune(parent string, unused time.Duration, keep string)
	Remove(folder string, along func() error) error
}
