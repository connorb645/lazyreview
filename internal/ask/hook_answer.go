package ask

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/utils"
)

const maxLinks = 40

// HookAnswer answers Claude Code's PreToolUse hook with the guard's
// decision on the call in input, as Claude Code writes it, for the copy of
// the code in folder copy. It returns the JSON that allows or refuses the
// call and the exit code. When the input cannot be read, it says why with
// code 2, which Claude Code takes as a refusal, so the guard never lets a
// call through by failing.
func HookAnswer(tr *i18n.TranslationSet, input []byte, copy string) (string, int) {
	var call HookCall
	if err := json.Unmarshal(input, &call); err != nil {
		return utils.ResolvePlaceholderString(tr.GuardFailed, map[string]string{"why": err.Error()}), 2
	}
	decision := Decide(tr, call, copy, realPath)
	output := map[string]any{"hookEventName": "PreToolUse", "permissionDecision": "deny"}
	if decision.Allow {
		output["permissionDecision"] = "allow"
	} else {
		output["permissionDecisionReason"] = decision.Refuse
	}
	text, err := json.Marshal(map[string]any{"hookSpecificOutput": output})
	if err != nil {
		return utils.ResolvePlaceholderString(tr.GuardFailed, map[string]string{"why": err.Error()}), 2
	}
	return string(text), 0
}

func realPath(path string) string {
	at, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return resolved(at, 0)
}

func resolved(path string, depth int) string {
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	at := string(filepath.Separator)
	for i, part := range parts[1:] {
		next := filepath.Join(at, part)
		info, err := os.Lstat(next)
		if err != nil {
			return filepath.Join(append([]string{next}, parts[i+2:]...)...)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			at = next
			continue
		}
		target, err := os.Readlink(next)
		if err != nil || depth >= maxLinks {
			return filepath.Join(append([]string{next}, parts[i+2:]...)...)
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(at, target)
		}
		at = resolved(target, depth+1)
	}
	return at
}
