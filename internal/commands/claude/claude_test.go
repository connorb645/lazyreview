package claude

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

const session = "3b241101-e2bb-4255-8caf-4136c566a962"

var settings = map[string]any{"hooks": map[string]any{"PreToolUse": []any{}}}

func TestMain(m *testing.M) {
	testsupport.ActAsFakeClaude()
	os.Exit(m.Run())
}

func folder(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestClaudeIsAskedOnceInPrintModeAsTheUsersOwnWithNothingThatNeedsApproval(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	fake := testsupport.NewFakeClaude(t)
	dir := folder(t)
	if err := os.WriteFile(filepath.Join(dir, "used.test.ts"), []byte("splitPence(1000, 3);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New(tr, fake.Command, "claude-sonnet-5-5")
	text, err := c.Ask(context.Background(), Asked{System: "Be brief.", Prompt: "Why?", Folder: dir, Session: session, Settings: settings}, nil)
	if err != nil || text != "It returns early when the list is empty." {
		t.Fatalf("the answer was %q, %v", text, err)
	}
	call := fake.Asked()[0]
	written, _ := json.Marshal(settings)
	want := []string{"-p", "--output-format", "stream-json", "--verbose", "--setting-sources", "user", "--settings", string(written),
		"--permission-mode", "default", "--permission-prompts", "none", "--append-system-prompt", "Be brief.", "--session-id", session, "--model", "claude-sonnet-5-5"}
	if call.Prompt != "Why?" || !reflect.DeepEqual(call.Args, want) {
		t.Errorf("Claude Code was run as %q with %q", call.Args, call.Prompt)
	}
	if call.Folder != dir || !reflect.DeepEqual(call.Files, map[string]string{"used.test.ts": "splitPence(1000, 3);\n"}) {
		t.Errorf("Claude Code ran in %s, reading %v", call.Folder, call.Files)
	}
	if _, err := c.Ask(context.Background(), Asked{System: "s", Prompt: "And?", Folder: dir, Session: session, Resume: true, Settings: settings}, nil); err != nil {
		t.Fatal(err)
	}
	if args := fake.Asked()[1].Args; args[14] != "--resume" || args[15] != session {
		t.Errorf("a follow-up was asked with %q", args)
	}
	_, err = c.Ask(context.Background(), Asked{System: "s", Prompt: "p", Folder: dir, Session: "00000000-0000-4000-8000-000000000000", Resume: true, Settings: settings}, nil)
	var lost Unanswered
	if !errors.As(err, &lost) || !lost.NoSession {
		t.Errorf("a session that is gone says %v", err)
	}
}

func TestEachToolCallIsHeardAsClaudeMakesItAndEachRefusalWithTheGuardsWords(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	fake := testsupport.NewFakeClaude(t)
	fake.Answer(testsupport.FakeAnswer{Text: "Only the test uses it.", Steps: []testsupport.FakeStep{
		{Tool: "Grep", Input: map[string]any{"pattern": "splitPence"}},
		{Tool: "Bash", Input: map[string]any{"command": "git push"}, Refused: "lazyreview refused this: it would push."},
		{Tool: "Bash", Input: map[string]any{"command": "rm -rf /"}, Refused: "Permission for this tool use was denied."},
		{Tool: "Bash", Input: map[string]any{"command": "echo x > ~/x"}, Refused: "Exit code 1\n(eval):1: operation not permitted: /Users/x/x"},
	}})
	var heard []Heard
	text, err := New(tr, fake.Command, "").Ask(context.Background(), Asked{System: "s", Prompt: "p", Folder: folder(t), Session: session, Settings: settings}, func(h Heard) { heard = append(heard, h) })
	if err != nil || text != "Only the test uses it." {
		t.Fatalf("the answer was %q, %v", text, err)
	}
	want := []Heard{
		{ID: "toolu_0", Tool: "Grep", Input: map[string]any{"pattern": "splitPence"}},
		{ID: "toolu_1", Tool: "Bash", Input: map[string]any{"command": "git push"}},
		{ID: "toolu_1", Refused: "it would push."},
		{ID: "toolu_2", Tool: "Bash", Input: map[string]any{"command": "rm -rf /"}},
		{ID: "toolu_2", Refused: "it needed approval, and nobody is there to give it"},
		{ID: "toolu_3", Tool: "Bash", Input: map[string]any{"command": "echo x > ~/x"}},
		{ID: "toolu_3", Refused: "the sandbox stopped it: shell commands change only the copy, and reach only the hosts allowed"},
	}
	if !reflect.DeepEqual(heard, want) {
		t.Errorf("heard %+v", heard)
	}
}

func TestStoppingClaudeEndsItAtOnceAndEveryWayItCanFailSaysWhy(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	fake := testsupport.NewFakeClaude(t)
	dir := folder(t)
	c := New(tr, fake.Command, "")
	ask := func(ctx context.Context, c *Claude) error {
		_, err := c.Ask(ctx, Asked{System: "s", Prompt: "p", Folder: dir, Session: session, Settings: settings}, nil)
		return err
	}
	fake.Answer(testsupport.FakeAnswer{Text: "Too late.", Wait: 30_000})
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(300*time.Millisecond, cancel)
	started := time.Now()
	if err := ask(ctx, c); err == nil || err.Error() != "you stopped it" || time.Since(started) > 10*time.Second {
		t.Errorf("a stop gave %v after %s", err, time.Since(started))
	}
	fake.Fail()
	if err := ask(context.Background(), c); err == nil || err.Error() != "Claude could not answer: the model is not available" {
		t.Errorf("a failure gave %v", err)
	}
	fake.Result(map[string]any{"subtype": "error_max_turns", "is_error": true, "num_turns": 40, "errors": []string{"Reached maximum number of turns (40)"}})
	if err := ask(context.Background(), c); err == nil || err.Error() != "Claude worked for 40 turns without answering; ask about less at once" {
		t.Errorf("too many turns gave %v", err)
	}
	if err := ask(context.Background(), New(tr, "/no/such/claude", "")); err == nil || err.Error() != "Claude Code did not answer: /no/such/claude was not found" {
		t.Errorf("a missing Claude Code gave %v", err)
	}
	if err := ask(context.Background(), New(tr, "false", "")); err == nil || err.Error() != "Claude Code did not answer: it stopped with exit code 1" {
		t.Errorf("a failing command gave %v", err)
	}
	c.limit = 300 * time.Millisecond
	fake.Answer(testsupport.FakeAnswer{Text: "Too late.", Wait: 30_000})
	if err := ask(context.Background(), c); err == nil || !strings.HasPrefix(err.Error(), "Claude did not answer within ") {
		t.Errorf("a slow answer gave %v", err)
	}
}
