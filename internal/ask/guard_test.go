package ask

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
)

const copyFolder = "/data/copies/github.com/octo/ledger/72/1"

func decided(tool string, input map[string]any) Decision {
	return Decide(i18n.EnglishTranslationSet(), HookCall{ToolName: tool, ToolInput: input, Cwd: copyFolder}, copyFolder, filepath.Clean)
}

func allowed(t *testing.T, tool string, input map[string]any) {
	t.Helper()
	if d := decided(tool, input); !d.Allow {
		t.Errorf("%s %v was refused: %s", tool, input, d.Refuse)
	}
}

func refused(t *testing.T, tool string, input map[string]any, words string) {
	t.Helper()
	if d := decided(tool, input); d.Allow || !regexp.MustCompile(words).MatchString(d.Refuse) {
		t.Errorf("%s %v gave %+v", tool, input, d)
	}
}

func TestReadingIsAllowedAndChangingTheCopyButNoFileOutsideIt(t *testing.T) {
	for _, tool := range []string{"Read", "Grep", "Glob", "WebFetch", "WebSearch", "Task", "Agent", "Skill", "TodoWrite", "ToolSearch", "ReadMcpResourceTool"} {
		allowed(t, tool, map[string]any{})
	}
	allowed(t, "Write", map[string]any{"file_path": copyFolder + "/src/split.ts"})
	allowed(t, "Edit", map[string]any{"file_path": "src/money.ts"})
	allowed(t, "NotebookEdit", map[string]any{"notebook_path": copyFolder + "/notes.ipynb"})
	refused(t, "Write", map[string]any{"file_path": "/etc/hosts"}, `^lazyreview refused this: the file is outside the copy of the code, and only the copy may change$`)
	refused(t, "Edit", map[string]any{"file_path": copyFolder + "/../2/src/money.ts"}, `outside the copy`)
	refused(t, "MultiEdit", map[string]any{"file_path": "/data/copies/github.com/octo/ledger/72/10/a.ts"}, `outside the copy`)
}

func TestALinkInsideTheCopyThatLeadsOutsideItIsOutsideEvenBeforeItsTargetExists(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	copy, elsewhere := filepath.Join(home, "copy"), filepath.Join(home, "elsewhere")
	for _, dir := range []string{copy, elsewhere} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(elsewhere, filepath.Join(copy, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(elsewhere, "not-yet.ts"), filepath.Join(copy, "dangling.ts")); err != nil {
		t.Fatal(err)
	}
	decision := func(path string) string {
		input, _ := json.Marshal(HookCall{ToolName: "Write", ToolInput: map[string]any{"file_path": path}, Cwd: copy})
		output, code := HookAnswer(tr, input, copy)
		var answer struct {
			HookSpecificOutput struct {
				PermissionDecision string `json:"permissionDecision"`
			} `json:"hookSpecificOutput"`
		}
		if err := json.Unmarshal([]byte(output), &answer); err != nil || code != 0 {
			t.Fatalf("the hook answered %q with %d", output, code)
		}
		return answer.HookSpecificOutput.PermissionDecision
	}
	if got := decision(filepath.Join(copy, "new", "file.ts")); got != "allow" {
		t.Errorf("a new file in the copy is %s", got)
	}
	if got := decision(filepath.Join(copy, "link", "file.ts")); got != "deny" {
		t.Errorf("a file through a link out of the copy is %s", got)
	}
	if got := decision(filepath.Join(copy, "dangling.ts")); got != "deny" {
		t.Errorf("a link out of the copy whose target is not there yet is %s", got)
	}
}

func TestShellCommandsRunButNotOneThatPushesPostsPublishesOrReachesAnotherMachine(t *testing.T) {
	for _, command := range []string{"npm test", "go test ./...", "git log --oneline | head", "git commit -am wip", `grep -rn "git push" .`, "curl -s https://example.com", "grep -rn open src"} {
		allowed(t, "Bash", map[string]any{"command": command})
	}
	refused(t, "Bash", map[string]any{"command": "git push origin HEAD:main"}, `^lazyreview refused this: it would push, and nobody is there to approve it$`)
	refused(t, "Bash", map[string]any{"command": "cd src && git -C .. push --force"}, `it would push,`)
	refused(t, "Bash", map[string]any{"command": "curl -X POST https://hooks.example.com -d text=hi"}, `it would send a change to a web service`)
	refused(t, "Bash", map[string]any{"command": `curl --json '{"a":1}' https://api.example.com`}, `web service`)
	refused(t, "Bash", map[string]any{"command": "http PATCH https://api.example.com/x a=1"}, `web service`)
	refused(t, "Bash", map[string]any{"command": "sudo -E npm publish"}, `publish a package`)
	refused(t, "Bash", map[string]any{"command": "docker push example/image"}, `publish a package`)
	refused(t, "Bash", map[string]any{"command": "echo ok; /usr/bin/ssh build.example.com"}, `reach a program or a machine`)
	refused(t, "Bash", map[string]any{"command": `claude -p "push it"`}, `reach a program or a machine`)
	refused(t, "Monitor", map[string]any{"command": "git push"}, `it would push,`)
}

func TestGhOnlyReadsOneCommandAtATime(t *testing.T) {
	for _, command := range []string{"gh pr view 72", "gh pr diff 72", "gh issue list", "gh api repos/octo/ledger/pulls/72", "gh api -X GET repos/octo/ledger", "gh search code splitPence"} {
		allowed(t, "Bash", map[string]any{"command": command})
	}
	for _, command := range []string{`gh pr comment 72 --body "Looks good"`, "gh pr review 72 --approve", "gh api repos/octo/ledger/issues/72/comments -f body=hi",
		"gh api --method POST repos/octo/ledger/issues", "gh pr view 72 && rm -rf ~", "gh pr view 72 > /tmp/x"} {
		refused(t, "Bash", map[string]any{"command": command}, `^lazyreview refused this: gh may only read`)
	}
}

func TestAnMCPToolIsAllowedOnlyWhenItsNameSaysItReadsAndAnUnknownToolIsRefused(t *testing.T) {
	for _, tool := range []string{"mcp__slack__slack_read_channel", "mcp__atlassian__getJiraIssue", "mcp__notes__search_notes", "mcp__datadog__get_datadog_metric"} {
		allowed(t, tool, map[string]any{})
	}
	refused(t, "mcp__slack__slack_send_message", map[string]any{}, `^lazyreview refused this: slack_send_message on slack may change something`)
	for _, tool := range []string{"mcp__atlassian__addCommentToJiraIssue", "mcp__notes__post_note", "mcp__jobs__check_job"} {
		refused(t, tool, map[string]any{}, `may change something`)
	}
	for _, tool := range []string{"PushNotification", "RemoteTrigger", "CronCreate", "SendMessage", "PublishPlugin", "SomethingNew"} {
		refused(t, tool, map[string]any{}, `is not allowed while nobody is there to approve it`)
	}
}

func TestTheGuardsSettingsTurnItsHookOnAndKeepShellCommandsInTheSandbox(t *testing.T) {
	settings := GuardSettings("'/app/lazyreview-terminal' guard '/copy'", []string{"example.com"})
	text, _ := json.Marshal(settings)
	var read map[string]any
	_ = json.Unmarshal(text, &read)
	want := map[string]any{
		"disableAllHooks": false, "enableAllProjectMcpServers": false,
		"hooks": map[string]any{"PreToolUse": []any{map[string]any{"matcher": "*", "hooks": []any{map[string]any{"type": "command", "command": "'/app/lazyreview-terminal' guard '/copy'"}}}}},
		"sandbox": map[string]any{"enabled": true, "autoAllowBashIfSandboxed": true, "allowUnsandboxedCommands": false, "failIfUnavailable": true, "excludedCommands": []any{"gh"},
			"network": map[string]any{"strictAllowlist": true, "allowedDomains": append(anyOf(Registries), "example.com")}},
	}
	if !reflect.DeepEqual(read, want) {
		t.Errorf("the settings are %s", text)
	}
}

func TestTheHookAnswersInClaudeCodesOwnWordsAndRefusesWhatItCannotRead(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	output, code := HookAnswer(tr, []byte(`{"tool_name":"Bash","tool_input":{"command":"git push"},"cwd":"/"}`), "/")
	if code != 0 || !strings.Contains(output, `"permissionDecision":"deny"`) {
		t.Errorf("a push was answered %q, %d", output, code)
	}
	if output, _ := HookAnswer(tr, []byte(`{"tool_name":"Read","tool_input":{},"cwd":"/"}`), "/"); output != `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}` {
		t.Errorf("a read was answered %q", output)
	}
	output, code = HookAnswer(tr, []byte("not json"), "/")
	if code != 2 || !strings.HasPrefix(output, "lazyreview refused this, as its guard could not read the call") {
		t.Errorf("a call it cannot read was answered %q, %d", output, code)
	}
}

func TestEachStepSaysWhatItDidNamingFilesFromTheCopysTop(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	folder := "/data/copies/x/1"
	cases := []struct {
		tool  string
		input map[string]any
		want  string
	}{
		{"Read", map[string]any{"file_path": "/data/copies/x/1/test/split.test.ts"}, "Read test/split.test.ts"},
		{"Grep", map[string]any{"pattern": "splitPence", "path": "/data/copies/x/1/src"}, "Searched for splitPence in src"},
		{"Glob", map[string]any{"pattern": "**/*.test.ts"}, "Looked for files named **/*.test.ts"},
		{"Bash", map[string]any{"command": "npm test\nnpm run lint"}, "Ran npm test"},
		{"Bash", map[string]any{"command": `cd "/data/copies/x/1" && gh pr view 72 > /data/copies/x/1/pr.txt`}, "Ran gh pr view 72 > pr.txt"},
		{"Write", map[string]any{"file_path": "/elsewhere/a.ts"}, "Wrote /elsewhere/a.ts"},
		{"WebSearch", map[string]any{"query": "splitting a bill fairly"}, "Searched the web for splitting a bill fairly"},
		{"Task", map[string]any{"description": "Find every caller"}, "Asked a helper to find every caller"},
		{"mcp__notes__post_note", map[string]any{}, "Used post_note on notes"},
	}
	for _, c := range cases {
		if got := StepWords(tr, c.tool, c.input, folder); got != c.want {
			t.Errorf("%s %v says %q", c.tool, c.input, got)
		}
	}
	if got := StepWords(tr, "Bash", map[string]any{"command": strings.Repeat("x", 400)}, folder); len([]rune(got)) != len("Ran ")+160 {
		t.Errorf("a long command is cut to %d", len([]rune(got)))
	}
}

func anyOf(list []string) []any {
	out := make([]any, len(list))
	for i, s := range list {
		out[i] = s
	}
	return out
}
