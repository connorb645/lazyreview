// Package git runs git for lazyreview's own repositories: the bare clones
// it keeps and the throwaway copies of the code Claude answers in.
package git

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"time"
)

const timeout = 15 * time.Minute

// Run runs git with args in dir, or where it was started with no dir, with
// env added to this process's environment and input on its standard input,
// until ctx ends.
type Run func(ctx context.Context, dir string, args []string, env []string, input []byte) Ran

// RunGit runs git itself, for fifteen minutes at most, never asking for a
// password on the terminal.
func RunGit(ctx context.Context, dir string, args []string, env []string, input []byte) Ran {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if dir != "" {
		args = append([]string{"-C", dir}, args...)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(append(os.Environ(), "GIT_TERMINAL_PROMPT=0"), env...)
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	done := Ran{Stdout: stdout.String(), Stderr: stderr.String()}
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		done.Code = exit.ExitCode()
	case err != nil:
		done.Code = 1
		if done.Stderr == "" {
			done.Stderr = err.Error()
		}
	}
	return done
}
