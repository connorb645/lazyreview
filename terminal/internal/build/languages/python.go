package languages

import (
	"regexp"
	"slices"
	"strings"
)

var (
	pythonClass     = regexp.MustCompile(`^([ \t]*)class[ \t]+([A-Za-z_]\w*)[ \t]*[(:\[]`)
	pythonDecorator = jsRegexp(`^\s*@`)
	pythonDataclass = regexp.MustCompile(`(?:^|\.)dataclass$`)
	pythonMetaclass = jsRegexp(`^metaclass\s*=`)
	pythonTestCase  = regexp.MustCompile(`TestCase$`)
	pythonInit      = jsRegexp(`^\s*(?:async\s+)?def\s+__init__\b`)
	pythonAnnotated = jsRegexp(`^\s*([A-Za-z_]\w*)\s*:\s*([^=\n]+)`)
	pythonStatement = regexp.MustCompile(`^(?:def|class|if|for|while|with|try|except|else|elif|return|async|match|case)$`)
	pythonTyped     = jsRegexp(`\bself\.(\w+)\s*:\s*([^=\n]+)`)
	pythonAssigned  = jsRegexp(`\bself\.(\w+)\s*=\s*([A-Z]\w*)\s*\(`)
	pythonFrom      = jsRegexp(`(?m)^[ \t]*from[ \t]+(\.*[\w.]*)[ \t]+import[ \t]+(\([^)]*\)|[^\n#;]+)`)
	pythonImport    = jsRegexp(`(?m)^[ \t]*import[ \t]+([^\n#;]+)`)
	pythonAs        = jsRegexp(`\s+as\s+`)
	pythonName      = regexp.MustCompile(`^\w+$`)
	pythonModule    = regexp.MustCompile(`^[\w.]+$`)

	pythonEnums  = []string{"Enum", "IntEnum", "StrEnum", "Flag", "IntFlag"}
	pythonLabels = map[string]string{"TypedDict": "typed dict", "NamedTuple": "named tuple", "ABC": "abstract class"}
)

func readPython(text string) Reading {
	masked := maskSource(text, pythonSyntax)
	lines := strings.Split(masked, "\n")
	shown := make([]string, len(lines))
	starts := make([]int, len(lines))
	offset := 0
	for i, line := range lines {
		starts[i] = offset
		shown[i] = unitBlanked(text, masked, offset, offset+len(line))
		offset += len(line) + 1
	}
	var found []Declaration
	for i, line := range shown {
		m := pythonClass.FindStringSubmatchIndex(line)
		if m == nil {
			continue
		}
		indent := len(strings.ReplaceAll(group(line, m, 1), "\t", "    "))
		k := starts[i] + m[1] - 1
		bases := []string{}
		if k < len(masked) && masked[k] == '[' {
			k = skipBalanced(masked, k)
		}
		if k < len(masked) && masked[k] == '(' {
			closing := skipBalanced(masked, k)
			bases = splitTopLevel(jsSlice(masked, k+1, closing-1))
			k = closing
		}
		headerEnd := lineOf(starts, indexFrom(masked, ":", k))
		last := headerEnd
		for j := headerEnd + 1; j < len(shown); j++ {
			if jsTrim(shown[j]) == "" {
				continue
			}
			if leadingJSSpace(strings.ReplaceAll(shown[j], "\t", "    ")) <= indent {
				break
			}
			last = j
		}
		var decorators []string
		for j := i - 1; j >= 0 && pythonDecorator.MatchString(shown[j]); j-- {
			decorators = append(decorators, strings.SplitN(jsTrim(shown[j])[1:], "(", 2)[0])
		}
		var named []string
		meta := ""
		for _, base := range bases {
			if !strings.Contains(base, "=") {
				named = append(named, base)
			} else if meta == "" && pythonMetaclass.MatchString(base) {
				meta = base
			}
		}
		inherits := bareNames(named)
		d := Declaration{
			Name:     group(line, m, 2),
			Kind:     pythonKind(inherits),
			Label:    pythonLabel(decorators, meta, inherits),
			Inherits: inherits,
			At:       starts[i] + m[3],
			Span:     [2]int{starts[i], starts[last] + len(lines[last])},
			Start:    i + 1,
			End:      last + 1,
			Test:     slices.ContainsFunc(inherits, pythonTestCase.MatchString),
		}
		d.Properties = pythonProperties(shown, headerEnd+1, last)
		found = append(found, d)
	}
	isTop := func(d Declaration) bool { return d.At == d.Span[0] }
	return Reading{Masked: masked, Declarations: nestDeclarations(found, isTop), Imports: pythonImports(masked)}
}

func pythonKind(inherits []string) string {
	switch {
	case slices.Contains(inherits, "Protocol"):
		return "protocol"
	case anyOf(inherits, pythonEnums):
		return "enum"
	}
	return "class"
}

func pythonLabel(decorators []string, meta string, inherits []string) *string {
	label := ""
	switch {
	case slices.ContainsFunc(decorators, pythonDataclass.MatchString):
		label = "dataclass"
	case strings.Contains(meta, "ABCMeta"):
		label = "abstract class"
	default:
		for _, base := range inherits {
			if pythonLabels[base] != "" {
				label = pythonLabels[base]
				break
			}
		}
	}
	if label == "" {
		return nil
	}
	return &label
}

func pythonProperties(shown []string, from, to int) []Property {
	out := []Property{}
	add := func(name, written, inferred string, line int) {
		if !slices.ContainsFunc(out, func(p Property) bool { return p.Name == name }) {
			out = append(out, storedProperty(name, written, inferred, line))
		}
	}
	bodyIndent, inInit, initIndent := -1, false, 0
	for j := from; j <= to; j++ {
		line := strings.ReplaceAll(shown[j], "\t", "    ")
		if jsTrim(line) == "" {
			continue
		}
		indent := leadingJSSpace(line)
		if bodyIndent < 0 {
			bodyIndent = indent
		}
		if indent == bodyIndent {
			inInit = pythonInit.MatchString(line)
			initIndent = indent
			if m := pythonAnnotated.FindStringSubmatch(line); m != nil && !pythonStatement.MatchString(m[1]) {
				add(m[1], jsTrim(m[2]), "", j+1)
			}
			continue
		}
		if !inInit || indent <= initIndent {
			continue
		}
		if m := pythonTyped.FindStringSubmatch(line); m != nil {
			add(m[1], jsTrim(m[2]), "", j+1)
		}
		if m := pythonAssigned.FindStringSubmatch(line); m != nil {
			add(m[1], "", m[2], j+1)
		}
	}
	return out
}

func pythonImports(masked string) []Import {
	var imports []Import
	text := strings.ReplaceAll(masked, "\\\n", " ")
	for _, m := range pythonFrom.FindAllStringSubmatch(text, -1) {
		found := Import{Spec: m[1], Python: true}
		if jsTrim(m[2]) != "*" {
			found.Names = []string{}
			for _, part := range strings.Split(strings.NewReplacer("(", " ", ")", " ").Replace(m[2]), ",") {
				pieces := pythonAs.Split(jsTrim(part), -1)
				if name := pieces[len(pieces)-1]; pythonName.MatchString(name) {
					found.Names = append(found.Names, name)
				}
			}
		}
		imports = append(imports, found)
	}
	for _, m := range pythonImport.FindAllStringSubmatch(text, -1) {
		for _, part := range strings.Split(m[1], ",") {
			if module := pythonAs.Split(jsTrim(part), -1)[0]; pythonModule.MatchString(module) {
				imports = append(imports, Import{Spec: module, Python: true})
			}
		}
	}
	return imports
}
