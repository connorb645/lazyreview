package ask

import (
	"context"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/commands/git"
)

// CodeCopies makes the throwaway copies of the code Claude answers in, and
// removes those no question used for a while.
type CodeCopies interface {
	Prepare(ctx context.Context, folder string, of git.CopyOf) (string, error)
	Prune(parent string, unused time.Duration, keep string)
}
