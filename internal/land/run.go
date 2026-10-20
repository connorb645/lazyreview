// Package land lands a branch on main as a fast-forward, on every remote
// and in the checkout it is run in. Nothing here forces a push or undoes
// one. It is this repository's own tool, run as make land BRANCH=<branch>,
// so its words are its own and not lazyreview's.
package land

import (
	"bytes"
	"os/exec"
	"strings"
)

// Run runs git with args in the checkout at repo.
type Run func(repo string, args []string) Ran

// RunGit runs git itself, with nothing on its standard input.
func RunGit(repo string, args []string) Ran {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err == nil {
		return Ran{OK: true, Out: strings.TrimSpace(stdout.String())}
	}
	why := stderr.String()
	if why == "" {
		why = err.Error()
	}
	return Ran{Out: strings.TrimSpace(stdout.String()), Err: strings.TrimSpace(why)}
}
