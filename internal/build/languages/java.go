package languages

import (
	"regexp"
	"strings"
)

var (
	javaDeclaration = jsRegexp(`(?:^|[\s;{}()])((?:@[\w.]+(?:\([^()\n]*\))?\s+)*)((?:(?:public|private|protected|static|final|abstract|sealed|non-sealed|strictfp)\s+)*)(class|interface|enum|record|@interface)\s+([A-Za-z_]\w*)`)
	javaField       = jsRegexp(`(?m)(?:^|[;{}])[ \t]*((?:@[\w.]+(?:\([^()\n]*\))?\s+)*)((?:(?:private|protected|public|static|final|transient|volatile)[ \t]+)*)([A-Z][\w.]*(?:<[^;=(){}]*>)?(?:\[\])*)[ \t]+(\w+)[ \t]*(?:=[ \t]*(?:new[ \t]+([A-Z]\w*))?|;)`)
	javaAnnotations = jsRegexp(`^(?:@[\w.]+(?:\([^()]*\))?\s+)*`)
	javaComponent   = jsRegexp(`^([\s\S]*\S)\s+(\w+)$`)
	javaNotTypes    = regexp.MustCompile(`^(?:return|throw|new|else|case)$`)
)

func readJava(text string) Reading {
	masked := maskSource(text, javaSyntax)
	index := indexSource(masked)
	var found []Declaration
	for _, m := range javaDeclaration.FindAllStringSubmatchIndex(masked, -1) {
		word := group(masked, m, 3)
		h, ok := javaHeader(masked, m[9], word)
		if !ok {
			continue
		}
		kind := word
		if word == "@interface" {
			kind = "annotation"
		}
		var label *string
		if hasWord(group(masked, m, 2), "abstract") && word == "class" {
			abstract := "abstract class"
			label = &abstract
		}
		closing := index.closing(h.open)
		d := Declaration{
			Name:     group(masked, m, 4),
			Kind:     kind,
			Label:    label,
			Inherits: bareNames(h.inherits),
			At:       m[6],
			Span:     [2]int{m[2], min(closing+1, len(masked))},
			Body:     &[2]int{h.open, closing},
			Depth:    index.depth(m[6]),
			Start:    index.line(m[2]),
			End:      index.line(closing),
			Test:     jvmTest.MatchString(text[h.open:closing]),
		}
		d.Properties = append(javaComponents(text, masked, index, h), javaFields(text, masked, index, d)...)
		found = append(found, d)
	}
	return Reading{Masked: masked, Declarations: nestDeclarations(found, atDepthZero)}
}

func javaHeader(masked string, k int, word string) (header, bool) {
	j := skipJSSpace(masked, k)
	if j < len(masked) && masked[j] == '<' {
		j = skipBalanced(masked, j)
	}
	h := header{parametersAt: -1}
	if word == "record" {
		j = skipJSSpace(masked, j)
		if j < len(masked) && masked[j] == '(' {
			closing := skipBalanced(masked, j)
			h.parametersAt = j + 1
			h.parameters = splitTopLevel(jsSlice(masked, j+1, closing-1))
			j = closing
		}
	}
	rest := masked[j:unitsAhead(masked, j, 4000)]
	brace := strings.IndexByte(rest, '{')
	if brace < 0 {
		return header{}, false
	}
	clause := rest[:brace]
	if strings.ContainsAny(clause, ";}=") {
		return header{}, false
	}
	stops := []string{"extends", "implements", "permits"}
	h.open = j + brace
	h.inherits = append(clausePart(clause, "extends", stops), clausePart(clause, "implements", stops)...)
	return h, true
}

func clausePart(clause, keyword string, stops []string) []string {
	at := wordAt(clause, keyword, 0)
	if at < 0 {
		return []string{}
	}
	from := at + len(keyword)
	to := len(clause)
	for _, stop := range stops {
		if next := wordAt(clause, stop, from); next >= 0 {
			to = min(to, next)
		}
	}
	return splitTopLevel(clause[from:to])
}

func javaComponents(text, masked string, index sourceIndex, h header) []Property {
	out := []Property{}
	cursor := h.parametersAt
	for _, component := range h.parameters {
		at := indexFrom(masked, component, cursor)
		cursor = at + len(component)
		bare := javaAnnotations.FindStringIndex(component)[1]
		m := javaComponent.FindStringSubmatchIndex(component[bare:])
		if m == nil || at < 0 {
			continue
		}
		written := shownType(text, masked, at+bare+m[2], at+bare+m[3])
		out = append(out, Property{Name: group(component[bare:], m, 2), Type: &written, Line: index.line(at)})
	}
	return out
}

func javaFields(text, masked string, index sourceIndex, d Declaration) []Property {
	out := []Property{}
	open := d.Body[0]
	inner := index.depth(open) + 1
	body := masked[open:d.Body[1]]
	for _, m := range javaField.FindAllStringSubmatchIndex(body, -1) {
		at := open + m[6]
		if index.depth(at) != inner || hasWord(group(body, m, 2), "static") || javaNotTypes.MatchString(group(body, m, 3)) {
			continue
		}
		written := shownType(text, masked, open+m[6], open+m[7])
		out = append(out, Property{Name: group(body, m, 4), Type: &written, Line: index.line(at)})
	}
	return out
}
