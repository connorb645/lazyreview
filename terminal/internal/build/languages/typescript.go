package languages

import (
	"regexp"
	"strings"
)

const tsQuoted = `(?:'([^'"\n]*)'|"([^'"\n]*)")`

var (
	tsDeclaration          = jsRegexp(`(?:^|[;{}\n])[ \t]*((?:@[\w.]+(?:\([^()]*\))?\s+)*)((?:(?:export|default|declare|abstract|const)\s+)*)(class|interface|enum|type|namespace|module)\s+([A-Za-z_$][\w$]*)`)
	tsField                = jsRegexp(`(?m)(?:^|[;{])[ \t]*((?:@[\w.]+(?:\([^()\n]*\))?\s+)*)((?:(?:public|private|protected|readonly|declare|override|abstract|static|accessor)[ \t]+)*)(#?[A-Za-z_$][\w$]*)[?!]?[ \t]*(?::[ \t]*([^=;\n]+?))?[ \t]*(?:=[ \t]*(?:new[ \t]+([A-Z][\w$]*))?[^;\n]*)?(;|\r|\x{2028}|\x{2029}|$)`)
	tsConstructor          = jsRegexp(`\bconstructor\s*\(`)
	tsConstructorParameter = jsRegexp(`^(?:(?:public|private|protected|readonly|override)\s+)+([A-Za-z_$][\w$]*)\??\s*:\s*([\s\S]+?)\s*(?:=[\s\S]*)?$`)
	tsImport               = jsRegexp(`(?m)(?:^|[;\n])[ \t]*(?:import|export)\s+(?:type\s+)?([\s\S]*?)\s*from\s*` + tsQuoted +
		`|(?:^|[;\n])[ \t]*import\s*` + tsQuoted + `|\brequire\s*\(\s*` + tsQuoted + `\s*\)|\bimport\s*\(\s*` + tsQuoted + `\s*\)`)
	tsEveryName  = jsRegexp(`^\s*\*`)
	tsBraces     = regexp.MustCompile(`\{((?s:.)*)\}`)
	tsImportName = jsRegexp(`^(?:type\s+)?([\w$]+)(?:\s+as\s+([\w$]+))?$`)
	tsHead       = regexp.MustCompile(`^[\w$]+$`)
)

func readTypeScript(text string) Reading {
	masked := maskSource(text, typescriptSyntax)
	index := indexSource(masked)
	var found, containers []Declaration
	for _, m := range tsDeclaration.FindAllStringSubmatchIndex(masked, -1) {
		word, keyword, start := group(masked, m, 3), m[6], m[2]
		var h header
		var ok bool
		if word == "type" {
			h, ok = tsAlias(masked, m[9])
		} else {
			h, ok = tsHeader(masked, m[9])
		}
		if !ok {
			continue
		}
		end := h.end
		var body *[2]int
		if h.open >= 0 {
			closing := index.closing(h.open)
			end, body = closing+1, &[2]int{h.open, closing}
		}
		var label *string
		if hasWord(group(masked, m, 2), "abstract") && word == "class" {
			abstract := "abstract class"
			label = &abstract
		}
		d := Declaration{
			Name:     group(masked, m, 4),
			Kind:     word,
			Label:    label,
			Inherits: bareNames(h.inherits),
			At:       keyword,
			Span:     [2]int{start, min(end, len(masked))},
			Body:     body,
			Depth:    index.depth(keyword),
			Start:    index.line(start),
			End:      index.line(max(keyword, end-1)),
		}
		if word == "namespace" || word == "module" {
			containers = append(containers, d)
			continue
		}
		d.Properties = tsProperties(text, masked, index, d)
		found = append(found, d)
	}
	insideContainersOnly := func(d Declaration) bool {
		count := 0
		for _, c := range containers {
			if c.Span[0] < d.At && d.At < c.Span[1] {
				count++
			}
		}
		return d.Depth == count
	}
	return Reading{Masked: masked, Declarations: nestDeclarations(found, insideContainersOnly), Imports: tsImports(text, masked)}
}

func tsHeader(masked string, k int) (header, bool) {
	j := skipJSSpace(masked, k)
	if j < len(masked) && masked[j] == '<' {
		j = skipBalanced(masked, j)
	}
	rest := masked[j:unitsAhead(masked, j, 4000)]
	brace := strings.IndexByte(rest, '{')
	if brace < 0 {
		return header{}, false
	}
	clause := rest[:brace]
	if strings.ContainsAny(strings.ReplaceAll(clause, "=>", ""), ";}=") {
		return header{}, false
	}
	stops := []string{"extends", "implements"}
	inherits := append(clausePart(clause, "extends", stops), clausePart(clause, "implements", stops)...)
	return header{open: j + brace, inherits: inherits}, true
}

func tsAlias(masked string, k int) (header, bool) {
	n := len(masked)
	j := skipJSSpace(masked, k)
	if j < n && masked[j] == '<' {
		j = skipBalanced(masked, j)
	}
	j = skipJSSpace(masked, j)
	if j >= n || masked[j] != '=' {
		return header{}, false
	}
	from := j + 1
	depth := 0
	for j = from; j < n; j++ {
		c := masked[j]
		if strings.IndexByte("([{<", c) >= 0 {
			depth++
		} else if strings.IndexByte(")]}", c) >= 0 || (c == '>' && masked[j-1] != '=') {
			depth = max(0, depth-1)
		} else if depth == 0 && c == ';' {
			break
		} else if depth == 0 && c == '\n' {
			next := jsTrimStart(masked[j+1 : unitsAhead(masked, j+1, 79)])
			before := jsTrimEnd(masked[from:j])
			if !(next != "" && strings.IndexByte("|&?:", next[0]) >= 0) && !(before != "" && strings.IndexByte("|&=:,(<{", before[len(before)-1]) >= 0) {
				break
			}
		}
	}
	return header{open: -1, end: j}, true
}

func tsProperties(text, masked string, index sourceIndex, d Declaration) []Property {
	out := []Property{}
	if d.Kind == "type" || d.Body == nil || d.Kind == "enum" {
		return out
	}
	open := d.Body[0]
	inner := index.depth(open) + 1
	body := masked[open:d.Body[1]]
	for _, m := range allBeforeLookahead(tsField, body, 6) {
		at := open + m[6]
		if index.depth(at) != inner || hasWord(group(body, m, 2), "static") {
			continue
		}
		written := ""
		if m[8] >= 0 {
			written = shownType(text, masked, open+m[8], open+m[9])
		}
		out = append(out, storedProperty(group(body, m, 3), written, group(body, m, 5), index.line(at)))
	}
	if constructor := tsConstructor.FindStringIndex(body); constructor != nil {
		start := open + constructor[1] - 1
		cursor := start + 1
		for _, parameter := range splitTopLevel(jsSlice(masked, start+1, skipBalanced(masked, start)-1)) {
			at := indexFrom(masked, parameter, cursor)
			cursor = at + len(parameter)
			m := tsConstructorParameter.FindStringSubmatchIndex(parameter)
			if m == nil || at < 0 {
				continue
			}
			written := shownType(text, masked, at+m[4], at+m[5])
			out = append(out, Property{Name: group(parameter, m, 1), Type: &written, Line: index.line(at)})
		}
	}
	return out
}

func allBeforeLookahead(re *regexp.Regexp, s string, lookahead int) [][]int {
	var out [][]int
	for pos := 0; pos <= len(s); {
		m := re.FindStringSubmatchIndex(s[pos:])
		if m == nil {
			break
		}
		for i := range m {
			if m[i] >= 0 {
				m[i] += pos
			}
		}
		m[1] = m[2*lookahead]
		out = append(out, m)
		pos = max(m[1], m[0]+1)
	}
	return out
}

func tsImports(text, masked string) []Import {
	var imports []Import
	for _, m := range tsImport.FindAllStringSubmatchIndex(masked, -1) {
		for g := 2; g <= 9; g++ {
			if m[2*g] < 0 {
				continue
			}
			found := Import{Spec: text[m[2*g]:m[2*g+1]]}
			if g <= 3 {
				found.Names = tsImportedNames(group(masked, m, 1))
			}
			imports = append(imports, found)
			break
		}
	}
	return imports
}

func tsImportedNames(clause string) []string {
	if tsEveryName.MatchString(clause) {
		return nil
	}
	names := []string{}
	braces := tsBraces.FindStringSubmatchIndex(clause)
	if braces != nil {
		for _, entry := range splitTopLevel(group(clause, braces, 1)) {
			if m := tsImportName.FindStringSubmatch(jsTrim(entry)); m != nil {
				if m[2] != "" {
					names = append(names, m[2])
				} else {
					names = append(names, m[1])
				}
			}
		}
		clause = clause[:braces[0]] + clause[braces[1]:]
	}
	head := jsTrim(strings.SplitN(clause, ",", 2)[0])
	if tsHead.MatchString(head) {
		names = append(names, head)
	}
	return names
}
