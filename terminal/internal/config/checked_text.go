package config

import (
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// CheckedText is a config as read written out as YAML, in the schema's
// order: every setting with where its value came from beside it, the
// default, the file or the variable that set it.
func CheckedText(tr *i18n.TranslationSet, read Read) string {
	return strings.Join(checkedLines(tr, Schema(tr), read.Values, read.Sources, "", ""), "\n") + "\n"
}

// CheckLines is what lazyreview config check prints: which config was
// read, every setting as CheckedText writes it, and each setting the file
// gives that does nothing now.
func CheckLines(tr *i18n.TranslationSet, read Read) []string {
	head := tr.ConfigUnset
	if read.Exists {
		head = tr.ConfigChecked
	}
	out := []string{utils.ResolvePlaceholderString(head, map[string]string{"path": read.Path}), "", jsTrimmed(CheckedText(tr, read))}
	if len(read.Unused) > 0 {
		out = append(out, "")
	}
	for _, key := range read.Unused {
		out = append(out, utils.ResolvePlaceholderString(tr.ConfigUnusedLine, map[string]string{"key": key, "why": Unused(tr)[key]}))
	}
	return out
}

// PathLines is what lazyreview config path prints: where the config file
// is, or would be, and whether it exists.
func PathLines(tr *i18n.TranslationSet, file ConfigFile) []string {
	if file.Exists {
		return []string{file.Path, tr.ConfigExists}
	}
	return []string{file.Path, tr.ConfigMissing}
}

func checkedLines(tr *i18n.TranslationSet, section *Section, values map[string]any, sources map[string]string, indent, prefix string) []string {
	var out []string
	for _, e := range section.Entries {
		key := e.Name
		if prefix != "" {
			key = prefix + "." + e.Name
		}
		if indent == "" && len(out) > 0 {
			out = append(out, "")
		}
		if e.Section != nil {
			head := indent + e.Name + ":"
			if e.Section.Note != "" {
				head = padded(head, noteColumn-1) + " # " + e.Section.Note
			}
			out = append(out, head)
			out = append(out, checkedLines(tr, e.Section, values[e.Name].(map[string]any), sources, indent+"  ", key)...)
			continue
		}
		value := values[e.Name]
		accounts, block := value.([]Account)
		block = block && len(accounts) > 0
		line := indent + e.Name + ":"
		switch {
		case block:
		case e.Setting.Check == "keys" || e.Setting.Check == "numberedKeys":
			line += " " + labels(value)
		default:
			line += " " + scalar(value)
		}
		out = append(out, padded(line, commentColumn-1)+" # "+sourceWords(tr, sources[key]))
		if block {
			for _, l := range strings.Split(jsTrimmed(dumpYAML(accountValues(accounts))), "\n") {
				out = append(out, indent+"  "+l)
			}
		}
	}
	return out
}

func sourceWords(tr *i18n.TranslationSet, source string) string {
	switch source {
	case "default":
		return tr.ConfigSourceDefault
	case "file":
		return tr.ConfigSourceFile
	}
	return source
}

func accountValues(accounts []Account) []any {
	out := make([]any, len(accounts))
	for i, a := range accounts {
		teams := make([]any, len(a.Teams))
		for j, t := range a.Teams {
			var since any
			if t.Since != nil {
				since = *t.Since
			}
			teams[j] = ordered("name", t.Name, "since", since)
		}
		out[i] = ordered("login", a.Login, "teams", teams)
	}
	return out
}

func jsTrimmed(text string) string {
	return strings.TrimRight(text, " \t\n\v\f\r\u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000\ufeff")
}
