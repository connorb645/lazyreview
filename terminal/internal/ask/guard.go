// Package ask asks Claude about lines of a review, with nobody there to
// approve anything: the guard on what Claude may do, the question it is
// asked, and the conversations that keep the questions and the answers.
package ask

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// Registries are the package registries shell commands may reach from the
// sandbox, so a test or a build can fetch what it needs.
var Registries = []string{
	"registry.npmjs.org", "registry.yarnpkg.com", "pypi.org", "files.pythonhosted.org", "proxy.golang.org", "sum.golang.org",
	"index.crates.io", "static.crates.io", "crates.io", "rubygems.org", "repo1.maven.org", "repo.maven.apache.org",
	"plugins.gradle.org", "services.gradle.org", "dl.google.com",
}

var reads = words("Read", "Grep", "Glob", "LS", "NotebookRead", "WebFetch", "WebSearch", "Task", "Agent", "Skill", "ToolSearch", "LSP",
	"TodoWrite", "TaskStop", "TaskOutput", "BashOutput", "KillShell", "ListAgents", "CronList", "ReportFindings",
	"ListMcpResourcesTool", "ReadMcpResourceTool", "ReadMcpResourceDirTool")

var fileChanges = map[string]string{"Write": "file_path", "Edit": "file_path", "MultiEdit": "file_path", "NotebookEdit": "notebook_path"}

var shells = map[string]string{"Bash": "command", "Monitor": "command"}

var readWords = words("get", "list", "search", "read", "fetch", "find", "query", "view", "describe", "show", "lookup", "inspect", "count", "browse", "retrieve")

var writeWords = words("create", "update", "delete", "remove", "post", "send", "write", "edit", "add", "set", "put", "patch", "upload", "publish",
	"share", "comment", "reply", "merge", "close", "assign", "move", "rename", "archive", "trash", "label", "mark", "approve", "submit", "cancel", "run",
	"execute", "trigger", "schedule", "save", "apply", "transition", "invite", "react", "resolve", "enable", "disable", "start", "stop", "generate",
	"import", "push", "sync", "insert", "modify", "change", "draft", "forward", "respond", "book", "pay", "deploy", "restore", "duplicate", "convert")

const commandStart = "(?:^|[;&|(`\\n]|\\$\\()\\s*(?:(?:sudo|env|exec|command|nohup|time|xargs)\\s+(?:-\\S+\\s+|\\w+=\\S*\\s+)*)*"

func run(programs, rest string) *regexp.Regexp {
	return regexp.MustCompile("(?i)" + commandStart + `(?:\S*/)?(?:` + programs + `)\b` + rest)
}

var outwards = []outward{
	{run("git", `[^;&|\n]*\s(push|send-email)\b`), "push"},
	{run("curl|wget", `[^;&|\n]*(\s-X\s*['"]?(POST|PUT|PATCH|DELETE)\b|\s--request\b|\s-d\b|\s--data|\s-F\b|\s--form\b|\s-T\b|\s--upload-file\b|\s--json\b|\s--post-|\s--method\b)`), "post"},
	{run("https?|xh", `\s+(POST|PUT|PATCH|DELETE)\b`), "post"},
	{run("npm|yarn|pnpm|bun|cargo|poetry|flit|twine|gem|docker|podman|helm", `[^;&|\n]*\s(publish|unpublish|upload|push)\b`), "publish"},
	{run("mvn|gradle|gradlew", `[^;&|\n]*\s(deploy|publish)\b`), "publish"},
	{run("ssh|scp|sftp|ftp|rsync|nc|ncat|telnet|osascript|open|claude", ""), "elsewhere"},
}

var (
	gh         = run("gh", "")
	ghReads    = regexp.MustCompile(`^\s*gh\s+((pr|issue|run|release|repo|workflow|gist|label|cache|ruleset|project)\s+(view|list|diff|checks|status)\b|search\s|auth\s+status\b|--version\b|api\s)`)
	ghChained  = regexp.MustCompile("[;&|<>`\\n]|\\$\\(")
	ghAPI      = regexp.MustCompile(`^\s*gh\s+api\s`)
	ghMethod   = regexp.MustCompile(`\s(-X|--method)[\s=]*['"]?(\w+)`)
	ghFields   = regexp.MustCompile(`\s(-f|-F|--field|--raw-field|--input)\b`)
	camelCase  = regexp.MustCompile(`([a-z])([A-Z])`)
	notLetters = regexp.MustCompile(`[^a-z]+`)
)

// GuardSettings are the settings lazyreview hands Claude Code over the
// user's own, which they outrank: hook, a command, is asked about every
// tool call, and hooks stay on whatever the user's settings say. Shell
// commands run in Claude Code's sandbox, with no way out of it: they write
// only in the copy and the temp folders, and reach the network only for
// domains, the package registries, and any host the user's own settings
// allow. gh runs outside the sandbox, so it can sign in, and the hook lets
// it only read.
func GuardSettings(hook string, domains []string) map[string]any {
	return map[string]any{
		"disableAllHooks":            false,
		"enableAllProjectMcpServers": false,
		"hooks": map[string]any{"PreToolUse": []any{
			map[string]any{"matcher": "*", "hooks": []any{map[string]any{"type": "command", "command": hook}}},
		}},
		"sandbox": map[string]any{
			"enabled": true, "autoAllowBashIfSandboxed": true, "allowUnsandboxedCommands": false, "failIfUnavailable": true, "excludedCommands": []string{"gh"},
			"network": map[string]any{"strictAllowlist": true, "allowedDomains": append(append([]string{}, Registries...), domains...)},
		},
	}
}

// Decide is the guard's answer to one tool call, for the copy of the code
// in folder copy, where real names where a path really leads, through any
// link. It allows tools that only read, file changes inside the copy,
// shell commands, which the sandbox holds in, and the tools of an MCP
// server whose names say they only read. It refuses a file change outside
// the copy, a shell command that pushes, posts, publishes or reaches
// another machine, gh doing anything but read, an MCP tool that may write,
// and any tool it does not know.
func Decide(tr *i18n.TranslationSet, call HookCall, copy string, real func(string) string) Decision {
	tool := call.ToolName
	if reads[tool] {
		return Decision{Allow: true}
	}
	if field, ok := fileChanges[tool]; ok {
		path := text(call.ToolInput[field])
		if !filepath.IsAbs(path) {
			base := call.Cwd
			if base == "" {
				base = copy
			}
			path = filepath.Join(base, path)
		}
		if inside(real(path), real(copy)) {
			return Decision{Allow: true}
		}
		return Decision{Refuse: tr.GuardOutside}
	}
	if field, ok := shells[tool]; ok {
		return shell(tr, text(call.ToolInput[field]))
	}
	if strings.HasPrefix(tool, "mcp__") {
		parts := strings.Split(tool, "__")
		server, name := parts[1], ""
		if len(parts) > 2 {
			name = parts[2]
		}
		split := notLetters.Split(strings.ToLower(camelCase.ReplaceAllString(name, "$1 $2")), -1)
		saysRead, saysWrite := false, false
		for _, w := range split {
			saysRead = saysRead || readWords[w]
			saysWrite = saysWrite || writeWords[w]
		}
		if saysRead && !saysWrite {
			return Decision{Allow: true}
		}
		return Decision{Refuse: utils.ResolvePlaceholderString(tr.GuardMcp, map[string]string{"server": server, "tool": name})}
	}
	return Decision{Refuse: utils.ResolvePlaceholderString(tr.GuardTool, map[string]string{"tool": tool})}
}

func shell(tr *i18n.TranslationSet, command string) Decision {
	if gh.MatchString(command) {
		if ghChained.MatchString(command) || !ghReads.MatchString(command) || !ghAPIReads(command) {
			return Decision{Refuse: tr.GuardGh}
		}
		return Decision{Allow: true}
	}
	for _, o := range outwards {
		if o.pattern.MatchString(command) {
			return Decision{Refuse: map[string]string{"push": tr.GuardPush, "post": tr.GuardPost, "publish": tr.GuardPublish, "elsewhere": tr.GuardElsewhere}[o.kind]}
		}
	}
	return Decision{Allow: true}
}

func ghAPIReads(command string) bool {
	if !ghAPI.MatchString(command) {
		return true
	}
	method := ghMethod.FindStringSubmatch(command)
	return (method == nil || strings.ToUpper(method[2]) == "GET") && !ghFields.MatchString(command)
}

func inside(path, folder string) bool {
	prefix := folder
	if !strings.HasSuffix(prefix, string(os.PathSeparator)) {
		prefix += string(os.PathSeparator)
	}
	return path == folder || strings.HasPrefix(path, prefix)
}

func text(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	}
	return fmt.Sprint(v)
}

func words(list ...string) map[string]bool {
	out := map[string]bool{}
	for _, w := range list {
		out[w] = true
	}
	return out
}
