package testsupport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const fakeClaudeHome = "LAZYREVIEW_FAKE_CLAUDE"

// FakeClaude is a stand-in for Claude Code, so tests ask no real model: the
// test program itself, run again as Command, answers one question in print
// mode with stream-json output, as Claude Code does. A session it has not
// saved cannot be resumed. A package whose tests run it calls
// ActAsFakeClaude first in its TestMain.
type FakeClaude struct {
	Command string
	home    string
	t       *testing.T
}

// NewFakeClaude makes the stand-in, answering "It returns early when the
// list is empty." until told otherwise.
func NewFakeClaude(t *testing.T) *FakeClaude {
	t.Helper()
	command, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	f := &FakeClaude{Command: command, home: t.TempDir(), t: t}
	t.Setenv(fakeClaudeHome, f.home)
	f.Answer(FakeAnswer{Text: "It returns early when the list is empty."})
	return f
}

// Answer sets what the stand-in answers next.
func (f *FakeClaude) Answer(answer FakeAnswer) {
	f.write(answer)
}

// Fail makes the stand-in fail, as Claude Code does when the model is not
// available.
func (f *FakeClaude) Fail() {
	f.write(nil)
}

// Result makes the stand-in print result as its result and fail, as Claude
// Code does when it stops early.
func (f *FakeClaude) Result(result map[string]any) {
	f.write(result)
}

// Knows makes the stand-in hold sessions, as if it had saved them, so a
// follow-up can go on from one.
func (f *FakeClaude) Knows(sessions ...string) {
	for _, s := range sessions {
		appendLine(filepath.Join(f.home, "sessions.txt"), s)
	}
}

// Asked is each run of the stand-in so far.
func (f *FakeClaude) Asked() []FakeCall {
	f.t.Helper()
	text, err := os.ReadFile(filepath.Join(f.home, "asked.jsonl"))
	if err != nil {
		return nil
	}
	var calls []FakeCall
	for _, line := range strings.Split(strings.TrimSpace(string(text)), "\n") {
		var call FakeCall
		if err := json.Unmarshal([]byte(line), &call); err != nil {
			f.t.Fatal(err)
		}
		calls = append(calls, call)
	}
	return calls
}

func (f *FakeClaude) write(v any) {
	f.t.Helper()
	text, err := json.Marshal(v)
	if err == nil {
		err = os.WriteFile(filepath.Join(f.home, "answer.json"), text, 0o644)
	}
	if err != nil {
		f.t.Fatal(err)
	}
}

// ActAsFakeClaude is the stand-in at work when a test ran this program as
// Claude Code, and returns at once when none did.
func ActAsFakeClaude() {
	home := os.Getenv(fakeClaudeHome)
	if home == "" {
		return
	}
	os.Exit(fakeClaude(home, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func fakeClaude(home string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	prompt, _ := io.ReadAll(stdin)
	folder, _ := os.Getwd()
	files := map[string]string{}
	_ = filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.Name() == ".git" && d.IsDir() {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			text, _ := os.ReadFile(path)
			rel, _ := filepath.Rel(folder, path)
			files[filepath.ToSlash(rel)] = string(text)
		}
		return nil
	})
	gitSays := func(a ...string) string {
		out, err := exec.Command("git", a...).Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	call := FakeCall{Args: args, Prompt: string(prompt), Folder: folder, Files: files,
		Git: FakeGit{Branch: gitSays("branch", "--show-current"), Branches: gitSays("branch", "--format=%(refname:short) %(objectname)"), Remotes: gitSays("remote")}}
	logged, _ := json.Marshal(call)
	appendLine(filepath.Join(home, "asked.jsonl"), string(logged))
	at := func(name string) string {
		if i := slices.Index(args, name); i >= 0 && i+1 < len(args) {
			return args[i+1]
		}
		return ""
	}
	known, _ := os.ReadFile(filepath.Join(home, "sessions.txt"))
	resume := slices.Contains(args, "--resume")
	if resume && !slices.Contains(strings.Split(string(known), "\n"), at("--resume")) {
		fmt.Fprintf(stderr, "No conversation found with session ID: %s\n", at("--resume"))
		return 1
	}
	session := at("--session-id")
	if resume {
		session = at("--resume")
	}
	appendLine(filepath.Join(home, "sessions.txt"), session)
	out := bufio.NewWriter(stdout)
	say := func(event map[string]any) {
		line, _ := json.Marshal(event)
		out.Write(append(line, '\n'))
		out.Flush()
	}
	say(map[string]any{"type": "system", "subtype": "init", "session_id": session, "cwd": folder, "tools": []string{"Read", "Bash"}})
	raw, _ := os.ReadFile(filepath.Join(home, "answer.json"))
	var spec map[string]any
	if err := json.Unmarshal(raw, &spec); err != nil || spec == nil {
		say(map[string]any{"type": "result", "subtype": "success", "is_error": true, "result": "the model is not available", "session_id": session})
		return 1
	}
	_, text := spec["text"]
	_, steps := spec["steps"]
	_, wait := spec["wait"]
	if !text && !steps && !wait {
		spec["type"], spec["session_id"] = "result", session
		say(spec)
		return 1
	}
	var answer FakeAnswer
	_ = json.Unmarshal(raw, &answer)
	for i, step := range answer.Steps {
		id := fmt.Sprintf("toolu_%d", i)
		input := step.Input
		if input == nil {
			input = map[string]any{}
		}
		say(map[string]any{"type": "assistant", "session_id": session, "message": map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "tool_use", "id": id, "name": step.Tool, "input": input}}}})
		content := "done"
		if step.Refused != "" {
			content = "PreToolUse:" + step.Tool + " hook error: " + step.Refused
		}
		say(map[string]any{"type": "user", "session_id": session, "message": map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "tool_result", "tool_use_id": id, "is_error": step.Refused != "", "content": content}}}})
	}
	time.Sleep(time.Duration(answer.Wait) * time.Millisecond)
	say(map[string]any{"type": "result", "subtype": "success", "is_error": false, "result": answer.Text, "session_id": session, "num_turns": 1})
	return 0
}

func appendLine(path, line string) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	fmt.Fprintln(file, line)
}
