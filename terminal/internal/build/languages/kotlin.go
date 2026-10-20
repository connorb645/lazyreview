package languages

import (
	"regexp"
	"strings"
)

const kotlinModifiers = `public|private|internal|protected|open|abstract|sealed|data|enum|annotation|inner|value|inline|expect|actual|final|fun|override`

var (
	kotlinDeclaration = jsRegexp(`(?:^|[\s;{}()])((?:@[\w.:]+(?:\([^()\n]*\))?\s+)*)((?:(?:` + kotlinModifiers + `)\s+)*)(class|interface|object)\s+([A-Za-z_]\w*)`)
	kotlinProperty    = jsRegexp(`(?m)(?:^|[;{])[ \t]*((?:@[\w.:]+(?:\([^()\n]*\))?\s+)*)((?:(?:public|private|internal|protected|open|abstract|override|final|lateinit|const|inline|external|actual|expect)[ \t]+)*)(val|var)[ \t]+(\w+)[ \t]*(?::[ \t]*([^=\n{]+))?(?:(?:=|by)[ \t]*(?:lazy[ \t]*\{[ \t]*)?([A-Z]\w*)[ \t]*[({])?`)
	kotlinParameter   = jsRegexp(`^(?:@[\w.:]+(?:\([^()]*\))?\s+)*(?:(?:private|public|internal|protected|override|open|final)\s+)*(val|var)\s+(\w+)\s*:\s*([\s\S]+?)\s*(?:=[\s\S]*)?$`)
	kotlinConstructor = jsRegexp(`^(?:(?:@[\w.:]+(?:\([^()\n]*\))?\s+)*(?:(?:private|public|internal|protected)\s+)?constructor\s*)?\(`)
	kotlinCompanion   = jsRegexp(`\bcompanion\s*$`)
	kotlinDelegation  = jsRegexp(`\s+by\s+[\s\S]*$`)
	kotlinContinues   = regexp.MustCompile(`^(?:[:,{]|where\b)`)
	kotlinAccessor    = jsRegexp(`\s(?:get|set)\b`)
	kotlinGetterHere  = jsRegexp(`^\s+get\b`)
	kotlinInitialised = regexp.MustCompile(`^[ \t]*(?:=|by\b)`)
	kotlinGetterBelow = regexp.MustCompile(`^[^\n]*\n[ \t]*(?:(?:private|protected|internal|public)[ \t]+)?get\b`)
	jvmTest           = regexp.MustCompile(`@(?:Test|ParameterizedTest)\b`)
)

func readKotlin(text string) Reading {
	masked := maskSource(text, kotlinSyntax)
	index := indexSource(masked)
	var found []Declaration
	for _, m := range kotlinDeclaration.FindAllStringSubmatchIndex(masked, -1) {
		start, modifiers, word := m[2], group(masked, m, 2), group(masked, m, 3)
		if word == "object" && kotlinCompanion.MatchString(masked[unitsBack(masked, start, 12):start]) {
			continue
		}
		h := kotlinHeader(masked, m[9])
		end := h.end
		var body *[2]int
		if h.open >= 0 {
			closing := index.closing(h.open)
			end, body = closing+1, &[2]int{h.open, closing}
		}
		supertypes := make([]string, len(h.inherits))
		for i, supertype := range h.inherits {
			if by := kotlinDelegation.FindStringIndex(supertype); by != nil {
				supertype = supertype[:by[0]]
			}
			supertypes[i] = supertype
		}
		d := Declaration{
			Name:     group(masked, m, 4),
			Kind:     kotlinKind(word, modifiers),
			Label:    kotlinLabel(word, modifiers),
			Inherits: bareNames(supertypes),
			At:       m[6],
			Span:     [2]int{start, min(end, len(masked))},
			Body:     body,
			Depth:    index.depth(m[6]),
			Start:    index.line(start),
			End:      index.line(max(start, end-1)),
			Test:     body != nil && jvmTest.MatchString(text[body[0]:body[1]]),
		}
		d.Properties = append(kotlinConstructorProperties(text, masked, index, h), kotlinProperties(text, masked, index, d)...)
		found = append(found, d)
	}
	return Reading{Masked: masked, Declarations: nestDeclarations(found, atDepthZero)}
}

func kotlinKind(word, modifiers string) string {
	switch {
	case word == "interface" || word == "object":
		return word
	case hasWord(modifiers, "enum"):
		return "enum"
	case hasWord(modifiers, "annotation"):
		return "annotation"
	}
	return "class"
}

func kotlinLabel(word, modifiers string) *string {
	label := ""
	switch {
	case word != "class" && hasWord(modifiers, "sealed"):
		label = "sealed " + word
	case word != "class" && hasWord(modifiers, "fun"):
		label = "fun " + word
	case word == "class":
		for _, special := range []string{"data", "sealed", "abstract", "value", "inline"} {
			if hasWord(modifiers, special) {
				if special == "inline" {
					special = "value"
				}
				label = special + " class"
				break
			}
		}
	}
	if label == "" {
		return nil
	}
	return &label
}

func kotlinHeader(masked string, k int) header {
	n := len(masked)
	j := k
	skipSpace := func() {
		for j < n && (masked[j] == ' ' || masked[j] == '\t') {
			j++
		}
	}
	skipSpace()
	if j < n && masked[j] == '<' {
		j = skipBalanced(masked, j)
		skipSpace()
	}
	h := header{open: -1, parametersAt: -1}
	if constructor := kotlinConstructor.FindStringIndex(masked[j:unitsAhead(masked, j, 400)]); constructor != nil {
		j += constructor[1] - 1
		closing := skipBalanced(masked, j)
		h.parametersAt = j + 1
		h.parameters = splitTopLevel(jsSlice(masked, j+1, closing-1))
		j = closing
	}
	var supertypes strings.Builder
	depth, collecting := 0, false
	for ; j < n; j++ {
		c := masked[j]
		if depth == 0 {
			if c == '{' {
				h.open = j
				break
			}
			if c == ';' || c == '}' || c == '=' {
				break
			}
			if c == '\n' {
				before := masked[k:j]
				if collecting {
					before = supertypes.String()
				}
				before = jsTrimEnd(before)
				after := jsTrimStart(masked[j+1 : unitsAhead(masked, j+1, 199)])
				if !strings.HasSuffix(before, ",") && !strings.HasSuffix(before, ":") && !kotlinContinues.MatchString(after) {
					break
				}
			}
			if c == ':' && !collecting {
				collecting = true
				continue
			}
		}
		if c == '(' || c == '<' || c == '[' {
			depth++
		} else if c == ')' || (c == '>' && masked[j-1] != '-') || c == ']' {
			depth = max(0, depth-1)
		}
		if collecting {
			supertypes.WriteByte(c)
		}
	}
	h.end = j
	written := supertypes.String()
	if where := wordAt(written, "where", 0); where >= 0 {
		written = written[:where]
	}
	h.inherits = splitTopLevel(written)
	return h
}

func kotlinConstructorProperties(text, masked string, index sourceIndex, h header) []Property {
	out := []Property{}
	cursor := h.parametersAt
	for _, parameter := range h.parameters {
		at := indexFrom(masked, parameter, cursor)
		cursor = at + len(parameter)
		m := kotlinParameter.FindStringSubmatchIndex(parameter)
		if m == nil || at < 0 {
			continue
		}
		written := shownType(text, masked, at+m[6], at+m[7])
		out = append(out, Property{Name: group(parameter, m, 2), Type: &written, Line: index.line(at)})
	}
	return out
}

func kotlinProperties(text, masked string, index sourceIndex, d Declaration) []Property {
	out := []Property{}
	if d.Body == nil {
		return out
	}
	open, closing := d.Body[0], d.Body[1]
	inner := index.depth(open) + 1
	body := masked[open:closing]
	for _, m := range kotlinProperty.FindAllStringSubmatchIndex(body, -1) {
		at := open + m[6]
		if index.depth(at) != inner || hasWord(group(body, m, 2), "const") {
			continue
		}
		typed := group(body, m, 5)
		accessor := kotlinAccessor.FindStringIndex(typed)
		from, to := open+m[10], open+m[11]
		if accessor != nil {
			to = from + accessor[0]
		}
		trimmed := ""
		if m[10] >= 0 {
			trimmed = jsTrim(masked[from:to])
		}
		after := open + m[9]
		if m[10] >= 0 {
			after = from + len(trimmed)
		}
		getterHere := accessor != nil && kotlinGetterHere.MatchString(typed[accessor[0]:])
		rest := masked[after:closing]
		initialised := !getterHere && kotlinInitialised.MatchString(rest)
		if getterHere || (kotlinGetterBelow.MatchString(rest) && !initialised) {
			continue
		}
		written := ""
		if trimmed != "" {
			written = shownType(text, masked, from, to)
		}
		out = append(out, storedProperty(group(body, m, 4), written, group(body, m, 6), index.line(at)))
	}
	return out
}
