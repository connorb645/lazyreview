package ask

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const long = 160

// StepWords says what one of Claude's tool calls does, in the words shown
// while Claude works and under its answer after: the tool's name, its input
// as Claude Code gives it, and folder, the copy of the code, whose paths it
// names from the copy's top. A command that starts by going there is named
// without it. Long words are cut.
func StepWords(tr *i18n.TranslationSet, tool string, input map[string]any, folder string) string {
	say := func(words, name, value string) string {
		return utils.ResolvePlaceholderString(words, map[string]string{name: value})
	}
	path := func(v any) string { return shown(text(v), folder) }
	switch tool {
	case "Read", "NotebookRead":
		return say(tr.StepRead, "path", path(first(input, "file_path", "notebook_path")))
	case "Grep":
		pattern := cut(text(input["pattern"]))
		if p := text(input["path"]); p != "" {
			return utils.ResolvePlaceholderString(tr.StepSearchedIn, map[string]string{"pattern": pattern, "path": path(p)})
		}
		return say(tr.StepSearched, "pattern", pattern)
	case "Glob":
		return say(tr.StepListed, "pattern", cut(text(input["pattern"])))
	case "Bash", "Monitor":
		return say(tr.StepRan, "command", cut(inCopy(strings.Split(text(input["command"]), "\n")[0], folder)))
	case "Edit", "MultiEdit", "NotebookEdit":
		return say(tr.StepEdited, "path", path(first(input, "file_path", "notebook_path")))
	case "Write":
		return say(tr.StepWrote, "path", path(input["file_path"]))
	case "WebFetch":
		return say(tr.StepFetched, "address", cut(text(input["url"])))
	case "WebSearch":
		return say(tr.StepSearchedWeb, "query", cut(text(input["query"])))
	case "Task", "Agent":
		return say(tr.StepHelper, "what", cut(strings.ToLower(text(first(input, "description", "prompt")))))
	case "Skill":
		return say(tr.StepSkill, "name", text(first(input, "skill", "name")))
	case "TodoWrite":
		return tr.StepPlanned
	}
	if strings.HasPrefix(tool, "mcp__") {
		parts := append(strings.Split(tool, "__"), "")
		return utils.ResolvePlaceholderString(tr.StepMcp, map[string]string{"tool": parts[2], "server": parts[1]})
	}
	return say(tr.StepUsed, "tool", tool)
}

func first(input map[string]any, names ...string) any {
	for _, n := range names {
		if v := text(input[n]); v != "" {
			return v
		}
	}
	return ""
}

func inCopy(command, folder string) string {
	q := regexp.QuoteMeta(folder)
	going := regexp.MustCompile(`^cd\s+(?:"` + q + `/?"|'` + q + `/?'|` + q + `/?)\s*(?:&&|;)\s*`)
	return strings.ReplaceAll(going.ReplaceAllString(command, ""), folder+"/", "")
}

func shown(path, folder string) string {
	if path == "" || !filepath.IsAbs(path) {
		return path
	}
	inside, err := filepath.Rel(folder, path)
	if err != nil || inside == "." || strings.HasPrefix(inside, "..") || filepath.IsAbs(inside) {
		return path
	}
	return filepath.ToSlash(inside)
}

func cut(text string) string {
	runes := []rune(text)
	if len(runes) > long {
		return string(runes[:long-1]) + "…"
	}
	return text
}
