package config

import (
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
)

const longestImplicitKey = 1024

var (
	base60          = regexp.MustCompile(`^[-+]?[0-9_]+(?::[0-9_]+)+(?:\.[0-9_]*)?$`)
	yamlDate        = regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]$`)
	yamlTimestamp   = regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]?-[0-9][0-9]?(?:[Tt]|[ \t]+)[0-9][0-9]?:[0-9][0-9]:[0-9][0-9](?:\.[0-9]*)?(?:[ \t]*(?:Z|[-+][0-9][0-9]?(?::[0-9][0-9])?))?$`)
	exponentNoPoint = regexp.MustCompile(`^[-+]?[0-9]+e`)
	oldBooleans     = []string{"y", "Y", "yes", "Yes", "YES", "on", "On", "ON", "n", "N", "no", "No", "NO", "off", "Off", "OFF"}
	escapes         = map[rune]string{0x00: `\0`, 0x07: `\a`, 0x08: `\b`, 0x09: `\t`, 0x0A: `\n`, 0x0B: `\v`, 0x0C: `\f`, 0x0D: `\r`, 0x1B: `\e`, 0x22: `\"`, 0x5C: `\\`, 0x85: `\N`, 0xA0: `\_`, 0x2028: `\L`, 0x2029: `\P`}
)

type scalarStyle int

const (
	plainStyle scalarStyle = iota
	singleStyle
	literalStyle
	doubleStyle
)

func dumpYAML(v any) string {
	return yamlNode(v, 0, true, true, false) + "\n"
}

func yamlNode(v any, level int, block, compact, isKey bool) string {
	switch x := v.(type) {
	case YAMLMap:
		if block && len(x.Keys) > 0 {
			return yamlBlockMapping(x, level, compact)
		}
		return yamlFlowMapping(x, level)
	case []any:
		if block && len(x) > 0 {
			return yamlBlockSequence(x, level, compact)
		}
		return yamlFlowSequence(x, level)
	case []string:
		items := make([]any, len(x))
		for i, s := range x {
			items[i] = s
		}
		return yamlNode(items, level, block, compact, isKey)
	case string:
		return yamlScalar(x, level, isKey, block)
	case nil:
		return "null"
	case bool:
		return strconv.FormatBool(x)
	case int:
		return strconv.Itoa(x)
	case float64:
		return yamlNumber(x)
	}
	return "null"
}

func yamlBlockMapping(m YAMLMap, level int, compact bool) string {
	var out strings.Builder
	for _, key := range m.Keys {
		if !compact || out.Len() > 0 {
			out.WriteString(nextLine(level))
		}
		name := yamlNode(key, level+1, true, true, true)
		explicit := len(utf16.Encode([]rune(name))) > longestImplicitKey
		if explicit {
			out.WriteString(marked("?", name))
		}
		out.WriteString(name)
		if explicit {
			out.WriteString(nextLine(level))
		}
		value := yamlNode(m.Values[key], level+1, true, explicit, false)
		out.WriteString(marked(":", value))
		out.WriteString(value)
	}
	if out.Len() == 0 {
		return "{}"
	}
	return out.String()
}

func yamlFlowMapping(m YAMLMap, level int) string {
	parts := make([]string, len(m.Keys))
	for i, key := range m.Keys {
		name := yamlNode(key, level, false, false, false)
		if len(utf16.Encode([]rune(name))) > longestImplicitKey {
			name = "? " + name
		}
		parts[i] = name + ": " + yamlNode(m.Values[key], level, false, false, false)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func yamlBlockSequence(items []any, level int, compact bool) string {
	var out strings.Builder
	for _, item := range items {
		written := yamlNode(item, level+1, true, true, false)
		if !compact || out.Len() > 0 {
			out.WriteString(nextLine(level))
		}
		out.WriteString(marked("-", written))
		out.WriteString(written)
	}
	if out.Len() == 0 {
		return "[]"
	}
	return out.String()
}

func yamlFlowSequence(items []any, level int) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = yamlNode(item, level, false, false, false)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func nextLine(level int) string {
	return "\n" + strings.Repeat("  ", level)
}

func marked(indicator, node string) string {
	if strings.HasPrefix(node, "\n") {
		return indicator
	}
	return indicator + " "
}

func yamlNumber(f float64) string {
	switch {
	case math.IsNaN(f):
		return ".nan"
	case math.IsInf(f, 1):
		return ".inf"
	case math.IsInf(f, -1):
		return "-.inf"
	case f == 0 && math.Signbit(f):
		return "-0.0"
	}
	text := jsNumber(f)
	if f != math.Trunc(f) && exponentNoPoint.MatchString(text) {
		return strings.Replace(text, "e", ".e", 1)
	}
	return text
}

func yamlScalar(s string, level int, isKey, inBlock bool) string {
	if s == "" {
		return "''"
	}
	if slices.Contains(oldBooleans, s) || base60.MatchString(s) {
		return "'" + s + "'"
	}
	switch scalarStyleOf(s, isKey, inBlock) {
	case plainStyle:
		return s
	case singleStyle:
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	case literalStyle:
		return "|" + blockHeader(s) + strings.TrimSuffix(indentedLines(s, 2*max(1, level)), "\n")
	}
	return `"` + escapedText(s) + `"`
}

func scalarStyleOf(s string, singleLine, inBlock bool) scalarStyle {
	runes := []rune(s)
	plain := plainFirst(runes[0]) && plainLast(runes[len(runes)-1])
	lineBreak := false
	previous := rune(-1)
	for _, c := range runes {
		switch {
		case c == '\n' && !singleLine:
			lineBreak = true
		case !printable(c):
			return doubleStyle
		}
		plain = plain && plainSafe(c, previous, inBlock)
		previous = c
	}
	if !lineBreak {
		if plain && !readsAsAnotherType(s) {
			return plainStyle
		}
		return singleStyle
	}
	return literalStyle
}

func printable(c rune) bool {
	return (0x20 <= c && c <= 0x7E) ||
		(0xA1 <= c && c <= 0xD7FF && c != 0x2028 && c != 0x2029) ||
		(0xE000 <= c && c <= 0xFFFD && c != 0xFEFF) ||
		(0x10000 <= c && c <= 0x10FFFF)
}

func whitespace(c rune) bool {
	return c == ' ' || c == '\t'
}

func nsCharOrWhitespace(c rune) bool {
	return printable(c) && c != 0xFEFF && c != '\r' && c != '\n'
}

func plainSafe(c, previous rune, inBlock bool) bool {
	nsOrWhite := nsCharOrWhitespace(c)
	ns := nsOrWhite && !whitespace(c)
	safe := nsOrWhite
	if !inBlock {
		safe = safe && !strings.ContainsRune(",[]{}", c)
	}
	return safe && c != '#' && !(previous == ':' && !ns) ||
		nsCharOrWhitespace(previous) && !whitespace(previous) && c == '#' ||
		previous == ':' && ns
}

func plainFirst(c rune) bool {
	return printable(c) && c != 0xFEFF && !whitespace(c) && !strings.ContainsRune("-?:,[]{}#&*!|=>'\"%@`", c)
}

func plainLast(c rune) bool {
	return !whitespace(c) && c != ':'
}

func readsAsAnotherType(s string) bool {
	_, isText := resolve(s).(string)
	return !isText || s == "<<" || yamlDate.MatchString(s) || yamlTimestamp.MatchString(s)
}

func blockHeader(s string) string {
	indicator := ""
	if strings.HasPrefix(strings.TrimLeft(s, "\n"), " ") {
		indicator = "2"
	}
	clip := strings.HasSuffix(s, "\n")
	switch {
	case clip && (strings.HasSuffix(s, "\n\n") || s == "\n"):
		return indicator + "+\n"
	case clip:
		return indicator + "\n"
	}
	return indicator + "-\n"
}

func indentedLines(s string, spaces int) string {
	var out strings.Builder
	for _, line := range strings.SplitAfter(s, "\n") {
		if line != "" && line != "\n" {
			out.WriteString(strings.Repeat(" ", spaces))
		}
		out.WriteString(line)
	}
	return out.String()
}

func escapedText(s string) string {
	var out strings.Builder
	for _, c := range s {
		escape, named := escapes[c]
		switch {
		case !named && printable(c):
			out.WriteRune(c)
		case named:
			out.WriteString(escape)
		case c <= 0xFF:
			out.WriteString(`\x` + hexDigits(c, 2))
		case c <= 0xFFFF:
			out.WriteString(`\u` + hexDigits(c, 4))
		default:
			out.WriteString(`\U` + hexDigits(c, 8))
		}
	}
	return out.String()
}

func hexDigits(c rune, width int) string {
	digits := strings.ToUpper(strconv.FormatInt(int64(c), 16))
	return strings.Repeat("0", width-len(digits)) + digits
}
