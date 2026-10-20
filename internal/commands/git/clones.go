package git

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/internal/i18n"
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

var credentials = []string{"-c", "credential.helper=", "-c", "credential.helper=!gh auth git-credential"}

// Prepare makes the bare clone at path the first time, fetching from url,
// then fetches the base branch and each pull request's head into it, and
// makes sure every commit in commits is there, fetching any missing one by
// its id. It returns the clone's path. Fetches go through gh as git's
// credential helper, as the account env names, so a private repository
// needs nothing set up beyond gh itself. A clone only ever holds what
// lazyreview fetched into it, so fetching overwrites its refs freely.
func (c *Clones) Prepare(ctx context.Context, path, url, base string, pulls []int, commits []string, env []string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		if err := c.git(ctx, "", []string{"init", "--quiet", "--bare", path}, nil); err != nil {
			return "", err
		}
		if err := c.git(ctx, path, []string{"remote", "add", "origin", url}, nil); err != nil {
			return "", err
		}
	} else if err := c.git(ctx, path, []string{"remote", "set-url", "origin", url}, nil); err != nil {
		return "", err
	}
	refspecs := []string{"+refs/heads/" + base + ":refs/heads/" + base}
	for _, n := range pulls {
		pull := strconv.Itoa(n)
		refspecs = append(refspecs, "+refs/pull/"+pull+"/head:refs/pull/"+pull+"/head")
	}
	fetch := append(append([]string{}, credentials...), "fetch", "--quiet", "--no-tags", "origin")
	if err := c.git(ctx, path, append(append([]string{}, fetch...), refspecs...), env); err != nil {
		return "", err
	}
	if missing := c.Missing(ctx, path, commits); len(missing) > 0 {
		if err := c.git(ctx, path, append(append([]string{}, fetch...), missing...), env); err != nil {
			return "", err
		}
	}
	return path, nil
}

// CopyFrom makes the bare clone at path the first time, then fetches each
// of commits it lacks from the repository in the folder from, which it
// only reads, keeping each under a ref of its own so it stays. It returns
// the clone's path.
func (c *Clones) CopyFrom(ctx context.Context, path, from string, commits []string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		if err := c.git(ctx, "", []string{"init", "--quiet", "--bare", path}, nil); err != nil {
			return "", err
		}
	}
	var refspecs []string
	for _, commit := range c.Missing(ctx, path, commits) {
		refspecs = append(refspecs, commit+":refs/commits/"+commit)
	}
	if len(refspecs) > 0 {
		if err := c.git(ctx, path, append([]string{"fetch", "--quiet", "--no-tags", from}, refspecs...), nil); err != nil {
			return "", err
		}
	}
	return path, nil
}

// Missing is each of commits the clone at path does not hold.
func (c *Clones) Missing(ctx context.Context, path string, commits []string) []string {
	var missing []string
	for _, commit := range commits {
		if c.run(ctx, path, []string{"cat-file", "-e", commit + "^{commit}"}, nil, nil).Code != 0 {
			missing = append(missing, commit)
		}
	}
	return missing
}

func (c *Clones) git(ctx context.Context, dir string, args, env []string) error {
	done := c.run(ctx, dir, args, env, nil)
	if done.Code != 0 {
		return failed(c.tr, commandOf(args), firstFailure(done.Stderr))
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
