package git

import (
	"context"
	"os"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

// Clones fetches into lazyreview's own bare clones of repositories. A clone
// only ever holds what lazyreview fetched into it. No call takes a lock:
// the caller keeps one fetch at a time in a clone.
type Clones struct {
	tr  *i18n.TranslationSet
	run Run
}

// NewClones fetches through run.
func NewClones(tr *i18n.TranslationSet, run Run) *Clones {
	if run == nil {
		run = RunGit
	}
	return &Clones{tr: tr, run: run}
}

// CopyFrom makes the bare clone at path the first time, then fetches each
// of commits it lacks from the repository in the folder from, which it
// only reads, keeping each under a ref of its own so it stays. It returns
// the clone's path.
func (c *Clones) CopyFrom(ctx context.Context, path, from string, commits []string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		if err := c.git(ctx, "", []string{"init", "--quiet", "--bare", path}); err != nil {
			return "", err
		}
	}
	var refspecs []string
	for _, commit := range commits {
		if c.run(ctx, path, []string{"cat-file", "-e", commit + "^{commit}"}, nil).Code != 0 {
			refspecs = append(refspecs, commit+":refs/commits/"+commit)
		}
	}
	if len(refspecs) > 0 {
		if err := c.git(ctx, path, append([]string{"fetch", "--quiet", "--no-tags", from}, refspecs...)); err != nil {
			return "", err
		}
	}
	return path, nil
}

func (c *Clones) git(ctx context.Context, dir string, args []string) error {
	done := c.run(ctx, dir, args, nil)
	if done.Code != 0 {
		return failed(c.tr, args[0], firstFailure(done.Stderr))
	}
	return nil
}

func firstFailure(stderr string) string {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "fatal:") || strings.HasPrefix(l, "error:") {
			return l
		}
	}
	return lines[len(lines)-1]
}
