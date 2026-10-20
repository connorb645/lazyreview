package languages

import (
	"regexp"
	"slices"
	"strings"
)

const (
	swiftAttribute = `@[\w.]+(?:\([^()\n]*\))?\s+`
	swiftModifier  = `(?:public|internal|private|fileprivate|open|final|indirect|nonisolated|package|dynamic|(?:public|internal|private|fileprivate|package)\(set\))\s+`
)

var (
	swiftDeclaration = jsRegexp(`(?:^|[\s;{}()])((?:` + swiftAttribute + `)*)((?:` + swiftModifier + `)*)(class|struct|enum|protocol|actor|extension)\s+([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*)`)
	swiftProperty    = jsRegexp(`(?m)(?:^|[;{])[ \t]*((?:@[\w.]+(?:\([^()\n]*\))?\s+)*)((?:(?:public|internal|private|fileprivate|open|final|weak|unowned(?:\(\w+\))?|lazy|nonisolated|static|class|override|dynamic|package|(?:public|internal|private|fileprivate|package)\(set\))[ \t]+)*)(let|var)[ \t]+(\w+)[ \t]*(?::[ \t]*([^={\n]+))?(?:=[ \t]*([A-Z]\w*)(?:\.\w+)*[ \t]*[({])?`)
	swiftCall        = jsRegexp(`^\s*\(`)
	swiftColon       = jsRegexp(`^\s*:`)
	swiftWhere       = jsRegexp(`^\s*where\b`)
	swiftObservers   = jsRegexp(`^\s*(?:willSet|didSet)\b`)
	swiftSuite       = regexp.MustCompile(`@Suite\b`)

	swiftNotNames = []string{"func", "var", "let", "subscript", "init", "deinit", "case", "typealias", "static", "class",
		"override", "convenience", "required", "mutating", "some", "any", "where", "protocol"}
	swiftViews = []string{"View", "App", "Scene", "ViewModifier", "Shape", "UIViewRepresentable",
		"UIViewControllerRepresentable", "NSViewRepresentable", "NSViewControllerRepresentable"}
	swiftTests = []string{"XCTestCase", "QuickSpec", "AsyncSpec"}
)

func readSwift(text string) Reading {
	masked := maskSource(text, swiftSyntax)
	index := indexSource(masked)
	var found []Declaration
	for _, m := range swiftDeclaration.FindAllStringSubmatchIndex(masked, -1) {
		name := group(masked, m, 4)
		if slices.Contains(swiftNotNames, name) {
			continue
		}
		h, ok := swiftHeader(masked, m[9])
		if !ok {
			continue
		}
		inherits := bareNames(h.inherits)
		closing := index.closing(h.open)
		d := Declaration{
			Name:     name,
			Kind:     group(masked, m, 3),
			Inherits: inherits,
			At:       m[6],
			Span:     [2]int{m[2], min(closing+1, len(masked))},
			Body:     &[2]int{h.open, closing},
			Depth:    index.depth(m[6]),
			Start:    index.line(m[2]),
			End:      index.line(closing),
			View:     anyOf(inherits, swiftViews),
			Test:     anyOf(inherits, swiftTests) || swiftSuite.MatchString(group(masked, m, 1)),
		}
		d.Properties = swiftProperties(text, masked, index, d)
		found = append(found, d)
	}
	return Reading{Masked: masked, Declarations: nestDeclarations(found, atDepthZero)}
}

func swiftHeader(masked string, k int) (header, bool) {
	j := k
	for j < len(masked) && (masked[j] == ' ' || masked[j] == '\t') {
		j++
	}
	if j < len(masked) && masked[j] == '<' {
		j = skipBalanced(masked, j)
	}
	rest := masked[j:unitsAhead(masked, j, 4000)]
	brace := strings.IndexByte(rest, '{')
	if brace < 0 {
		return header{}, false
	}
	clause := rest[:brace]
	if strings.ContainsAny(clause, ";}=") || swiftCall.MatchString(clause) {
		return header{}, false
	}
	inherits := ""
	if colon := swiftColon.FindStringIndex(clause); colon != nil {
		inherits = clause[colon[1]:]
		if where := wordAt(inherits, "where", 0); where >= 0 {
			inherits = inherits[:where]
		}
	} else if jsTrim(clause) != "" && !swiftWhere.MatchString(clause) {
		return header{}, false
	}
	return header{open: j + brace, inherits: splitTopLevel(inherits)}, true
}

func swiftProperties(text, masked string, index sourceIndex, d Declaration) []Property {
	out := []Property{}
	if d.Kind == "extension" || d.Kind == "protocol" {
		return out
	}
	open := d.Body[0]
	inner := index.depth(open) + 1
	body := masked[open:d.Body[1]]
	for _, m := range swiftProperty.FindAllStringSubmatchIndex(body, -1) {
		at := open + m[6]
		if index.depth(at) != inner || hasWord(group(body, m, 2), "static", "class") {
			continue
		}
		end := m[9]
		if m[10] >= 0 {
			end = m[11]
		}
		if m[12] < 0 && swiftComputed(masked, index, open+end) {
			continue
		}
		written := ""
		if m[10] >= 0 {
			written = shownType(text, masked, open+m[10], open+m[11])
		}
		out = append(out, storedProperty(group(body, m, 4), written, group(body, m, 6), index.line(at)))
	}
	return out
}

func swiftComputed(masked string, index sourceIndex, end int) bool {
	limit := unitsAhead(masked, end, 200)
	j := end
	for j < limit && (masked[j] == ' ' || masked[j] == '\t') {
		j++
	}
	if j >= limit || masked[j] != '{' {
		return false
	}
	return !swiftObservers.MatchString(masked[j+1 : index.closing(j)])
}

func anyOf(names, wanted []string) bool {
	return slices.ContainsFunc(names, func(name string) bool { return slices.Contains(wanted, name) })
}
