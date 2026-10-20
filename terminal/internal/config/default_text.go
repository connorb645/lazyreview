package config

import (
	"regexp"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

const (
	commentColumn = 39
	noteColumn    = 26
)

var (
	plainScalar = regexp.MustCompile(`^[A-Za-z0-9~/][A-Za-z0-9~/._-]*$`)
	specialWord = regexp.MustCompile(`(?i)^(true|false|null|~|[0-9][0-9._]*)$`)
)

// DefaultConfigText is the default config written out as YAML, in the
// schema's order, with the words saying what each setting does and,
// beside it, the variable that overrides it: what lazyreview config
// defaults prints and docs/Config.md shows.
func DefaultConfigText(tr *i18n.TranslationSet) string {
	return strings.Join(defaultLines(Schema(tr), ""), "\n") + "\n"
}

func defaultLines(section *Section, indent string) []string {
	var out []string
	for i, e := range section.Entries {
		if indent == "" && len(out) > 0 {
			out = append(out, "")
		}
		if e.Section != nil && e.Section.Flow {
			var parts []string
			for _, inner := range e.Section.Entries {
				parts = append(parts, inner.Name+": "+scalar(inner.Setting.Default))
			}
			out = append(out, indent+e.Name+": {"+strings.Join(parts, ", ")+"}")
			continue
		}
		if e.Section != nil {
			head := indent + e.Name + ":"
			if e.Section.Note != "" {
				head = padded(head, noteColumn-1) + " # " + e.Section.Note
			}
			out = append(out, head)
			for _, line := range e.Section.About {
				out = append(out, indent+"  # "+line)
			}
			out = append(out, defaultLines(e.Section, indent+"  ")...)
			continue
		}
		s := e.Setting
		if i > 0 && section.Entries[i-1].Setting != nil && len(section.Entries[i-1].Setting.Example) > 0 {
			out = append(out, "")
		}
		for _, line := range s.About {
			out = append(out, indent+"# "+line)
		}
		written := scalar(s.Default)
		if s.Check == "keys" || s.Check == "numberedKeys" {
			written = labels(s.Default)
		}
		line := indent + e.Name + ": " + written
		if s.Env != "" {
			line = padded(line, commentColumn-1) + " # " + s.Env
		}
		out = append(out, line)
		for _, example := range s.Example {
			out = append(out, indent+"# "+example)
		}
	}
	return out
}

func padded(text string, width int) string {
	if n := len([]rune(text)); n < width {
		return text + strings.Repeat(" ", width-n)
	}
	return text
}

func labels(v any) string {
	quoted := func(label string) string { return "'" + strings.ReplaceAll(label, "'", "''") + "'" }
	if list, ok := v.([]string); ok {
		parts := make([]string, len(list))
		for i, l := range list {
			parts[i] = quoted(l)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return quoted(v.(string))
}

func scalar(v any) string {
	switch x := v.(type) {
	case []string:
		parts := make([]string, len(x))
		for i, item := range x {
			parts[i] = scalar(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case []Account:
		return "[]"
	case string:
		if plainScalar.MatchString(x) && !specialWord.MatchString(x) {
			return x
		}
		return "'" + strings.ReplaceAll(x, "'", "''") + "'"
	}
	return jsText(v)
}
