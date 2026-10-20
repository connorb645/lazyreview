// Package claude asks Claude one question, through Claude Code run once in
// print mode as the user's own: their settings, CLAUDE.md, skills, MCP
// servers and default model, with every tool.
package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/utils"
)

const (
	timeout = 30 * time.Minute
	grace   = 3 * time.Second
)

var (
	noSession  = regexp.MustCompile(`(?i)no conversation found`)
	notAllowed = regexp.MustCompile(`(?i)permission .*denied|was denied`)
	sandboxed  = regexp.MustCompile(`(?i)<sandbox_violations>|operation not permitted`)
)

// Claude runs Claude Code. Nothing that would ask for approval runs, and no
// settings come from the folder it works in, as the code there may be
// anyone's. Each question has a time limit.
type Claude struct {
	tr      *i18n.TranslationSet
	command string
	model   string
	limit   time.Duration
}

// New asks through command, Claude Code, with model over the user's
// default when one is named.
func New(tr *i18n.TranslationSet, command, model string) *Claude {
	if command == "" {
		command = "claude"
	}
	return &Claude{tr: tr, command: command, model: model, limit: timeout}
}

// Ask asks one question, telling heard each tool call as Claude makes it
// and each one refused, with why: the guard's words after its mark, that it
// needed approval, or that the sandbox stopped it, even part of a command
// that went on. Ending ctx stops Claude Code and everything it started. It
// returns the answer's text, or an Unanswered saying why there is none.
func (c *Claude) Ask(ctx context.Context, asked Asked, heard func(Heard)) (string, error) {
	settings, err := json.Marshal(asked.Settings)
	if err != nil {
		return "", err
	}
	args := []string{"-p", "--output-format", "stream-json", "--verbose", "--setting-sources", "user", "--settings", string(settings),
		"--permission-mode", "default", "--permission-prompts", "none", "--append-system-prompt", asked.System}
	if asked.Resume {
		args = append(args, "--resume", asked.Session)
	} else {
		args = append(args, "--session-id", asked.Session)
	}
	if c.model != "" {
		args = append(args, "--model", c.model)
	}
	cmd := exec.Command(c.command, args...)
	cmd.Dir = asked.Folder
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdin = strings.NewReader(asked.Prompt)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		why := err.Error()
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, syscall.ENOENT) {
			why = utils.ResolvePlaceholderString(c.tr.ClaudeNotFound, map[string]string{"command": c.command})
		}
		return "", c.unanswered(c.tr.ClaudeNoAnswer, why, false)
	}
	var (
		mu     sync.Mutex
		why    string
		exited bool
	)
	stop := func(reason string) {
		mu.Lock()
		defer mu.Unlock()
		if why != "" || exited {
			return
		}
		why = reason
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		time.AfterFunc(grace, func() {
			mu.Lock()
			defer mu.Unlock()
			if !exited {
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
		})
	}
	limit := time.AfterFunc(c.limit, func() { stop("slow") })
	defer limit.Stop()
	watching := make(chan struct{})
	defer close(watching)
	go func() {
		select {
		case <-ctx.Done():
			stop("stopped")
		case <-watching:
		}
	}()
	result := c.read(stdout, heard)
	waited := cmd.Wait()
	mu.Lock()
	exited = true
	reason := why
	mu.Unlock()
	switch {
	case reason == "stopped":
		return "", Unanswered{Why: c.tr.ClaudeStopped}
	case reason == "slow":
		return "", Unanswered{Why: utils.ResolvePlaceholderString(c.tr.ClaudeTooSlow, map[string]string{"minutes": strconv.Itoa(int(c.limit / time.Minute))})}
	case result == nil:
		lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
		last := lines[len(lines)-1]
		if last == "" {
			last = utils.ResolvePlaceholderString(c.tr.ClaudeExitCode, map[string]string{"code": strconv.Itoa(exitCode(waited))})
		}
		return "", c.unanswered(c.tr.ClaudeNoAnswer, last, noSession.MatchString(stderr.String()))
	case result.Subtype == "error_max_turns":
		return "", Unanswered{Why: utils.ResolvePlaceholderString(c.tr.ClaudeTooManyTurns, map[string]string{"turns": strconv.Itoa(result.NumTurns)})}
	case result.IsError || result.Result == nil:
		why := ""
		if result.Result != nil {
			why = *result.Result
		}
		if why == "" {
			why = strings.Join(result.Errors, "; ")
		}
		if why == "" {
			why = result.Subtype
		}
		if why == "" {
			why = c.tr.ClaudeNothing
		}
		words := utils.ResolvePlaceholderString(c.tr.ClaudeCouldNotAnswer, map[string]string{"why": why})
		return "", Unanswered{Why: words, NoSession: noSession.MatchString(words)}
	}
	return strings.TrimSpace(*result.Result), nil
}

func (c *Claude) read(stdout io.Reader, heard func(Heard)) *event {
	var result *event
	lines := bufio.NewReader(stdout)
	for {
		line, err := lines.ReadString('\n')
		var e event
		if json.Unmarshal([]byte(line), &e) == nil {
			if e.Type == "result" {
				result = &e
			} else {
				c.hear(e, heard)
			}
		}
		if err != nil {
			return result
		}
	}
}

func (c *Claude) hear(e event, heard func(Heard)) {
	if heard == nil {
		return
	}
	for _, b := range e.Message.Content {
		if e.Type == "assistant" && b.Type == "tool_use" {
			input := b.Input
			if input == nil {
				input = map[string]any{}
			}
			heard(Heard{ID: b.ID, Tool: b.Name, Input: input})
		}
		if e.Type != "user" || b.Type != "tool_result" {
			continue
		}
		text := resultText(b.Content)
		at := strings.Index(text, c.tr.GuardMarker)
		switch {
		case b.IsError && at >= 0:
			rest := text[at+len(c.tr.GuardMarker):]
			if strings.HasPrefix(rest, ":") || strings.HasPrefix(rest, ",") {
				rest = strings.TrimLeftFunc(rest[1:], unicode.IsSpace)
			}
			heard(Heard{ID: b.ToolUseID, Refused: strings.TrimSpace(rest)})
		case b.IsError && notAllowed.MatchString(text):
			heard(Heard{ID: b.ToolUseID, Refused: c.tr.ClaudeNotApproved})
		case sandboxed.MatchString(text):
			heard(Heard{ID: b.ToolUseID, Refused: c.tr.ClaudeSandboxed})
		}
	}
}

func resultText(content json.RawMessage) string {
	var text string
	if json.Unmarshal(content, &text) == nil {
		return text
	}
	var parts []json.RawMessage
	if json.Unmarshal(content, &parts) != nil {
		var one struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(content, &one)
		return one.Text
	}
	var out []string
	for _, p := range parts {
		var s string
		if json.Unmarshal(p, &s) == nil {
			out = append(out, s)
			continue
		}
		var one struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(p, &one)
		out = append(out, one.Text)
	}
	return strings.Join(out, "\n")
}

func (c *Claude) unanswered(words, why string, session bool) Unanswered {
	return Unanswered{Why: utils.ResolvePlaceholderString(words, map[string]string{"why": why}), NoSession: session}
}

func exitCode(err error) int {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	if err != nil {
		return 1
	}
	return 0
}
