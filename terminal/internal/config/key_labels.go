package config

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// Disabled alone is no key.
const Disabled = "<disabled>"

var (
	namedKeys = []string{"space", "enter", "esc", "tab", "up", "down", "left", "right"}
	ctrlKey   = regexp.MustCompile(`^c-([a-z])$`)
)

// ReaderName is a key sequence as the config writes it, as lazygit writes
// its own, the way the terminal app's key reader names it: its keys joined
// by spaces, such as "g g", "space v" or "ctrl+o", or "" for <disabled>. A
// terminal sends Tab for Ctrl-i, so <c-i> is "tab".
func ReaderName(label string) string {
	if label == Disabled {
		return ""
	}
	var out []string
	for _, token := range labelTokens(label) {
		name, isNamed := named(token)
		switch {
		case !isNamed:
			out = append(out, token)
		case ctrlKey.MatchString(name):
			letter := ctrlKey.FindStringSubmatch(name)[1]
			if letter == "i" {
				out = append(out, "tab")
			} else {
				out = append(out, "ctrl+"+letter)
			}
		default:
			out = append(out, name)
		}
	}
	return strings.Join(out, " ")
}

// LabelProblem says what is wrong with a label, in the config's words, or
// "" when nothing is.
func LabelProblem(tr *i18n.TranslationSet, label any) string {
	text, ok := label.(string)
	switch {
	case !ok:
		return tr.ConfigKeyNotText
	case text == Disabled:
		return ""
	case text == "":
		return tr.ConfigKeyEmpty
	}
	for _, token := range labelTokens(text) {
		if token == "<" {
			return tr.ConfigKeyUnclosed
		}
		name, isNamed := named(token)
		if !isNamed {
			continue
		}
		if name == "disabled" {
			return tr.ConfigKeyDisabledAlone
		}
		known := ctrlKey.MatchString(name)
		for _, n := range namedKeys {
			known = known || n == name
		}
		if !known {
			return utils.ResolvePlaceholderString(tr.ConfigKeyUnknown, map[string]string{"token": token})
		}
	}
	return ""
}

func named(token string) (string, bool) {
	if len(token) > 2 && strings.HasPrefix(token, "<") && strings.HasSuffix(token, ">") {
		return token[1 : len(token)-1], true
	}
	return "", false
}

func labelTokens(label string) []string {
	var out []string
	for i := 0; i < len(label); {
		if label[i] == '<' {
			if end := strings.IndexByte(label[i:], '>'); end > 1 {
				out = append(out, label[i:i+end+1])
				i += end + 1
				continue
			}
		}
		_, size := utf8.DecodeRuneInString(label[i:])
		out = append(out, label[i:i+size])
		i += size
	}
	return out
}
