package config

import (
	"errors"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var (
	plainItem    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/*-]*$`)
	specialItem  = regexp.MustCompile(`(?i)^(true|false|null|~|[0-9][0-9._]*)$`)
	leadingSpace = regexp.MustCompile(`^ *`)
	itemLine     = regexp.MustCompile(`^-(\s|$)`)
)

// ListEdited is config.yml's text with one list setting changed in place,
// as a person would edit it, so every other line stays as it was: comments,
// blank lines, commented-out settings, the order and every value. key
// names a section's list, such as "github.hiddenRepositories". Each item
// to add the list lacks goes at its end, and each item to remove comes
// out. A list the text lacks is written at the end of its section, and a
// section it lacks at the end of the text, so empty text gives a file
// holding only the list. Items keep the list's own style: one a line, or
// all on one line in brackets. It says why when the text is not YAML, or
// when the list or its section is written in a way it cannot change line
// by line, such as brackets over several lines; and it reads what it
// wrote, refusing unless that holds the list asked for and every other
// value as it was.
func ListEdited(tr *i18n.TranslationSet, text, key string, change ListChange) (string, error) {
	sectionName, name, _ := strings.Cut(key, ".")
	cannot := errors.New(utils.ResolvePlaceholderString(tr.ConfigCannotChange, map[string]string{"key": key}))
	parsed, err := ParseYAML(tr, text)
	if err != nil {
		return "", err
	}
	was, err := listIn(parsed, sectionName, name, cannot)
	if err != nil {
		return "", err
	}
	var wanted []string
	for _, item := range was {
		if !slices.Contains(change.Remove, item) {
			wanted = append(wanted, item)
		}
	}
	for _, item := range change.Add {
		if !slices.Contains(was, item) && !slices.Contains(wanted, item) {
			wanted = append(wanted, item)
		}
	}
	if slices.Equal(was, wanted) {
		return text, nil
	}
	lines := strings.Split(text, "\n")
	ending := ""
	if strings.Contains(text, "\r\n") {
		ending = "\r"
	}
	step := indentUnit(lines)
	header := -1
	headerPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(sectionName) + `\s*:(\s|$)`)
	for i, line := range lines {
		if header < 0 && headerPattern.MatchString(bare(line)) {
			header = i
		}
	}
	if header < 0 {
		block := []string{sectionName + ":", strings.Repeat(" ", step) + name + ":"}
		for _, item := range wanted {
			block = append(block, strings.Repeat(" ", step*2)+"- "+plainYAML(item))
		}
		lines = appendSection(lines, block, ending)
	} else {
		if valueAfterColon(bare(lines[header])) != "" {
			return "", errors.New(utils.ResolvePlaceholderString(tr.ConfigCannotChange, map[string]string{"key": sectionName}))
		}
		end := sectionEnd(lines, header)
		child := step
		for _, line := range lines[header+1 : end] {
			if isContent(line) {
				child = indentOf(line)
				break
			}
		}
		namePattern := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(name) + `\s*:(\s|$)`)
		at := -1
		for i := header + 1; i < end; i++ {
			if at < 0 && isContent(lines[i]) && indentOf(lines[i]) == child && namePattern.MatchString(bare(lines[i])) {
				at = i
			}
		}
		if at < 0 {
			after := header
			for i := header + 1; i < end; i++ {
				if !isBlank(lines[i]) && indentOf(lines[i]) > 0 {
					after = i
				}
			}
			block := []string{strings.Repeat(" ", child) + name + ":" + ending}
			for _, item := range wanted {
				block = append(block, strings.Repeat(" ", child+step)+"- "+plainYAML(item)+ending)
			}
			lines = slices.Insert(lines, after+1, block...)
		} else {
			value := valueAfterColon(bare(lines[at]))
			switch {
			case strings.HasPrefix(value, "["):
				edited, err := flowEdited(tr, lines[at], wanted, change.Remove, cannot)
				if err != nil {
					return "", err
				}
				lines[at] = edited
			case value != "":
				return "", cannot
			default:
				if lines, err = blockEdited(tr, lines, at, child, step, wanted, change.Remove, ending, cannot); err != nil {
					return "", err
				}
			}
		}
	}
	out := strings.Join(lines, "\n")
	after, err := ParseYAML(tr, out)
	if err != nil {
		return "", cannot
	}
	now, err := listIn(after, sectionName, name, cannot)
	if err != nil || !slices.Equal(now, wanted) || !reflect.DeepEqual(without(parsed, sectionName, name), without(after, sectionName, name)) {
		return "", cannot
	}
	return out, nil
}

func listIn(value any, sectionName, name string, cannot error) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	top, ok := value.(YAMLMap)
	if !ok {
		return nil, cannot
	}
	held := top.Values[sectionName]
	if held == nil {
		return nil, nil
	}
	section, ok := held.(YAMLMap)
	if !ok {
		return nil, cannot
	}
	list := section.Values[name]
	if list == nil {
		return nil, nil
	}
	items, ok := list.([]any)
	if !ok {
		return nil, cannot
	}
	out := []string{}
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, cannot
		}
		out = append(out, text)
	}
	return out, nil
}

// without is a value as plain maps, with the list taken out, and its
// section too when nothing else is in it, so two values compare by what
// they hold, whatever their keys' order.
func without(value any, sectionName, name string) any {
	plain := unordered(value)
	top, ok := plain.(map[string]any)
	if !ok {
		return plain
	}
	if section, ok := top[sectionName].(map[string]any); ok {
		delete(section, name)
		if len(section) == 0 {
			delete(top, sectionName)
		}
	} else if top[sectionName] == nil {
		delete(top, sectionName)
	}
	return top
}

func unordered(v any) any {
	switch x := v.(type) {
	case YAMLMap:
		out := map[string]any{}
		for _, k := range x.Keys {
			out[k] = unordered(x.Values[k])
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = unordered(item)
		}
		return out
	case nil:
		return map[string]any{}
	}
	return v
}

func sectionEnd(lines []string, header int) int {
	for i := header + 1; i < len(lines); i++ {
		if isContent(lines[i]) && indentOf(lines[i]) == 0 {
			return i
		}
	}
	return len(lines)
}

func blockEdited(tr *i18n.TranslationSet, lines []string, at, child, step int, wanted, remove []string, ending string, cannot error) ([]string, error) {
	end := len(lines)
	for i := at + 1; i < len(lines); i++ {
		if isContent(lines[i]) && (indentOf(lines[i]) < child || (indentOf(lines[i]) == child && !isItem(lines[i]))) {
			end = i
			break
		}
	}
	var items []int
	for i := at + 1; i < end; i++ {
		if !isContent(lines[i]) {
			continue
		}
		if !isItem(lines[i]) || (len(items) > 0 && indentOf(lines[i]) != indentOf(lines[items[0]])) {
			return nil, cannot
		}
		items = append(items, i)
	}
	indent := child + step
	if len(items) > 0 {
		indent = indentOf(lines[items[0]])
	}
	value := func(i int) any {
		v, _ := ParseYAML(tr, strings.TrimLeft(bare(lines[i]), " \t")[1:])
		return v
	}
	removed := map[int]bool{}
	var kept []int
	held := map[any]bool{}
	for _, i := range items {
		if v, ok := value(i).(string); ok && slices.Contains(remove, v) {
			removed[i] = true
			continue
		}
		kept = append(kept, i)
		held[value(i)] = true
	}
	var added []string
	for _, item := range wanted {
		if !held[item] {
			added = append(added, strings.Repeat(" ", indent)+"- "+plainYAML(item)+ending)
		}
	}
	last := at
	if len(kept) > 0 {
		last = kept[len(kept)-1]
	}
	var out []string
	for i, line := range lines {
		if !removed[i] {
			out = append(out, line)
		}
		if i == last {
			out = append(out, added...)
		}
	}
	return out, nil
}

func flowEdited(tr *i18n.TranslationSet, line string, wanted, remove []string, cannot error) (string, error) {
	text := bare(line)
	open := strings.Index(text[strings.Index(text, ":"):], "[") + strings.Index(text, ":")
	close := closingBracket(text, open)
	if close < 0 || valueAfterColon(":"+text[close+1:]) != "" {
		return "", cannot
	}
	var raw []string
	held := map[any]bool{}
	for _, item := range splitItems(text[open+1 : close]) {
		v, _ := ParseYAML(tr, item)
		if s, ok := v.(string); ok && slices.Contains(remove, s) {
			continue
		}
		raw = append(raw, item)
		held[v] = true
	}
	for _, item := range wanted {
		if !held[item] {
			raw = append(raw, plainYAML(item))
		}
	}
	suffix := ""
	if strings.HasSuffix(line, "\r") {
		suffix = "\r"
	}
	return text[:open] + "[" + strings.Join(raw, ", ") + "]" + text[close+1:] + suffix, nil
}

func closingBracket(text string, open int) int {
	quote := byte(0)
	for i := open + 1; i < len(text); i++ {
		ch := text[i]
		switch {
		case quote == '"' && ch == '\\':
			i++
		case quote != 0 && ch == quote:
			quote = 0
		case quote == 0 && (ch == '"' || ch == '\''):
			quote = ch
		case quote == 0 && (ch == '[' || ch == '{'):
			return -1
		case quote == 0 && ch == ']':
			return i
		}
	}
	return -1
}

func splitItems(inside string) []string {
	var items []string
	quote := byte(0)
	from := 0
	for i := 0; i <= len(inside); i++ {
		ch := byte(0)
		if i < len(inside) {
			ch = inside[i]
		}
		switch {
		case quote == '"' && ch == '\\':
			i++
		case quote != 0 && ch == quote:
			quote = 0
		case quote == 0 && (ch == '"' || ch == '\''):
			quote = ch
		case quote == 0 && (ch == ',' || i == len(inside)):
			if item := strings.TrimSpace(inside[from:i]); item != "" {
				items = append(items, item)
			}
			from = i + 1
		}
	}
	return items
}

func appendSection(lines, block []string, ending string) []string {
	if lines[len(lines)-1] != "" {
		lines[len(lines)-1] += ending
		lines = append(lines, "")
	}
	var spaced []string
	if len(lines) > 1 && !isBlank(lines[len(lines)-2]) {
		spaced = []string{ending}
	}
	var withEndings []string
	for _, line := range block {
		withEndings = append(withEndings, line+ending)
	}
	return slices.Insert(lines, len(lines)-1, append(spaced, withEndings...)...)
}

func indentUnit(lines []string) int {
	unit := 0
	for _, line := range lines {
		if n := indentOf(line); isContent(line) && n > 0 && (unit == 0 || n < unit) {
			unit = n
		}
	}
	if unit == 0 {
		return 2
	}
	return unit
}

func valueAfterColon(text string) string {
	rest := strings.TrimSpace(text[strings.Index(text, ":")+1:])
	if strings.HasPrefix(rest, "#") {
		return ""
	}
	return rest
}

func plainYAML(item string) string {
	if plainItem.MatchString(item) && !specialItem.MatchString(item) {
		return item
	}
	return "'" + strings.ReplaceAll(item, "'", "''") + "'"
}

func bare(line string) string { return strings.TrimSuffix(line, "\r") }

func indentOf(line string) int { return len(leadingSpace.FindString(line)) }

func isBlank(line string) bool { return strings.TrimSpace(bare(line)) == "" }

func isComment(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(bare(line), " \t"), "#")
}

func isContent(line string) bool { return !isBlank(line) && !isComment(line) }

func isItem(line string) bool { return itemLine.MatchString(strings.TrimLeft(bare(line), " \t")) }
