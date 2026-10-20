package git

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

var quiet = []string{"-c", "core.hooksPath=/dev/null", "-c", "advice.detachedHead=false"}

var copyEnv = []string{"GIT_NO_LAZY_FETCH=1", "GIT_LFS_SKIP_SMUDGE=1"}

// Copies makes throwaway copies of a repository at a commit, for Claude
// Code to work in. A copy borrows its clone's objects instead of copying
// them, and has no remote, so a push has nowhere to go. Making one runs no
// hook and fetches nothing. Neither Prepare nor Prune takes a lock: the
// caller keeps one at a time under the copies' folder.
type Copies struct {
	tr  *i18n.TranslationSet
	run Run
	now func() time.Time
}

// NewCopies makes copies through run, marking them used at now.
func NewCopies(tr *i18n.TranslationSet, run Run, now func() time.Time) *Copies {
	if run == nil {
		run = RunGit
	}
	if now == nil {
		now = time.Now
	}
	return &Copies{tr: tr, run: run, now: now}
}

// Prepare makes folder a copy of what of describes, unless an earlier call
// did, marks it used now and returns it. The folder appears whole or not
// at all.
func (c *Copies) Prepare(ctx context.Context, folder string, of CopyOf) (string, error) {
	if _, err := os.Stat(filepath.Join(folder, ".git")); err != nil {
		if err := c.make(ctx, folder, of); err != nil {
			return "", err
		}
	}
	now := c.now()
	return folder, os.Chtimes(folder, now, now)
}

// Prune removes each copy under parent not used for unused, other than
// keep, and the folders it leaves empty.
func (c *Copies) Prune(parent string, unused time.Duration, keep string) {
	before := c.now().Add(-unused)
	var walk func(folder string)
	walk = func(folder string) {
		entries, err := os.ReadDir(folder)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			path := filepath.Join(folder, entry.Name())
			if path == keep {
				continue
			}
			if _, err := os.Stat(filepath.Join(path, ".git")); err == nil || strings.HasPrefix(entry.Name(), ".") {
				if info, err := os.Stat(path); err == nil && info.ModTime().Before(before) {
					_ = os.RemoveAll(path)
				}
				continue
			}
			walk(path)
			if left, err := os.ReadDir(path); err == nil && len(left) == 0 {
				_ = os.Remove(path)
			}
		}
	}
	walk(parent)
}

func (c *Copies) make(ctx context.Context, folder string, of CopyOf) error {
	if c.run(ctx, of.Source, []string{"cat-file", "-e", of.Commit + "^{commit}"}, nil, nil).Code != 0 {
		return models.Refused(utils.ResolvePlaceholderString(c.tr.CommitMissing, map[string]string{"commit": short(of.Commit), "repo": of.Source}))
	}
	if err := os.MkdirAll(filepath.Dir(folder), 0o755); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(filepath.Dir(folder), "."+filepath.Base(folder)+"-")
	if err != nil {
		return err
	}
	made := func() error {
		if err := c.git(ctx, "", append(append([]string{}, quiet...), "clone", "--quiet", "--shared", "--no-checkout", of.Source, temporary)); err != nil {
			return err
		}
		if err := c.git(ctx, temporary, []string{"remote", "remove", "origin"}); err != nil {
			return err
		}
		names := make([]string, 0, len(of.Branches))
		for name := range of.Branches {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if err := c.git(ctx, temporary, []string{"update-ref", "refs/heads/" + name, of.Branches[name]}); err != nil {
				return err
			}
		}
		if err := c.git(ctx, temporary, append(append([]string{}, quiet...), "checkout", "--quiet", of.Branch)); err != nil {
			return err
		}
		if err := os.RemoveAll(folder); err != nil {
			return err
		}
		return os.Rename(temporary, folder)
	}()
	if made != nil {
		_ = os.RemoveAll(temporary)
	}
	return made
}

func (c *Copies) git(ctx context.Context, dir string, args []string) error {
	done := c.run(ctx, dir, args, copyEnv, nil)
	if done.Code != 0 {
		return failed(c.tr, commandOf(args), lastLine(done.Stderr))
	}
	return nil
}

func commandOf(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") && !strings.Contains(a, "=") {
			return a
		}
	}
	return ""
}

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	return lines[len(lines)-1]
}

func failed(tr *i18n.TranslationSet, command, why string) error {
	return models.Refused(utils.ResolvePlaceholderString(tr.GitFailed, map[string]string{"command": command, "why": why}))
}

func short(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}
