package languages

import (
	"regexp"
	"strings"
)

var regexAfterWord = regexp.MustCompile(`(?:^|[^\w$])(?:return|typeof|case|do|else|in|of|new|delete|void|throw|yield|await)$`)

type masker struct {
	text   string
	syntax syntax
}

func maskSource(text string, s syntax) string {
	m := masker{text: text, syntax: s}
	out := []byte(text)
	for k := 0; k < len(text); {
		found, ok := m.literalAt(k)
		if !ok {
			k++
			continue
		}
		from, to := found.inner[0], found.inner[1]
		if found.comment {
			from, to = found.start, found.end
		}
		for i := from; i < min(to, len(out)); i++ {
			if out[i] != '\n' {
				out[i] = ' '
			}
		}
		k = max(found.end, k+1)
	}
	return string(out)
}

func (m masker) at(token string, k int) bool {
	return k <= len(m.text) && strings.HasPrefix(m.text[k:], token)
}

func (m masker) literalAt(k int) (literal, bool) {
	for _, marker := range m.syntax.line {
		if m.at(marker, k) {
			return literal{start: k, end: m.lineEnd(k), comment: true}, true
		}
	}
	for _, pair := range m.syntax.block {
		if m.at(pair[0], k) {
			return literal{start: k, end: m.blockEnd(k, pair[0], pair[1]), comment: true}, true
		}
	}
	hashes := 0
	if m.syntax.rawHashes {
		for k+hashes < len(m.text) && m.text[k+hashes] == '#' {
			hashes++
		}
	}
	lead := 0
	if m.syntax.prefixLetters != "" && hashes == 0 && (k == 0 || !isWordByte(m.text[k-1])) {
		lead = m.prefixAt(k)
	}
	for _, spec := range m.syntax.literals {
		if m.at(spec.open, k+lead+hashes) {
			return m.stringAt(k, spec, hashes, lead), true
		}
	}
	if m.syntax.regex && m.text[k] == '/' && !m.at("//", k) && !m.at("/*", k) && m.regexAllowedBefore(k) {
		return m.regexAt(k)
	}
	return literal{}, false
}

func (m masker) prefixAt(k int) int {
	for size := 2; size >= 1; size-- {
		if k+size >= len(m.text) || (m.text[k+size] != '"' && m.text[k+size] != '\'') {
			continue
		}
		letters := true
		for i := k; i < k+size; i++ {
			letters = letters && strings.IndexByte(m.syntax.prefixLetters, m.text[i]) >= 0
		}
		if letters {
			return size
		}
	}
	return 0
}

func (m masker) lineEnd(k int) int {
	for k < len(m.text) && m.text[k] != '\n' {
		k++
	}
	return k
}

func (m masker) blockEnd(k int, open, close string) int {
	depth := 1
	k += len(open)
	for k < len(m.text) {
		if m.syntax.nestedBlocks && m.at(open, k) {
			depth++
			k += len(open)
			continue
		}
		if m.at(close, k) {
			k += len(close)
			depth--
			if depth == 0 {
				return k
			}
			continue
		}
		k++
	}
	return len(m.text)
}

func (m masker) codeEnd(k int, open, close byte) int {
	depth := 1
	for k < len(m.text) {
		if found, ok := m.literalAt(k); ok {
			k = found.end
			continue
		}
		if m.text[k] == open {
			depth++
		} else if m.text[k] == close {
			depth--
			if depth == 0 {
				return k + 1
			}
		}
		k++
	}
	return len(m.text)
}

func (m masker) stringAt(k int, spec literalSyntax, hashes, lead int) literal {
	n := len(m.text)
	innerStart := k + lead + hashes + len(spec.open)
	close := spec.close + strings.Repeat("#", hashes)
	escape := spec.escape
	if hashes > 0 {
		escape = `\` + strings.Repeat("#", hashes)
	}
	for j := innerStart; j < n; {
		if !spec.multiline && m.text[j] == '\n' {
			return literal{start: k, end: j, inner: [2]int{innerStart, j}}
		}
		if escape != "" && m.at(escape, j) {
			next := j + len(escape)
			if spec.interpolation == `\(` && next < n && m.text[next] == '(' {
				j = m.codeEnd(next+1, '(', ')')
				continue
			}
			j = next + 1
			continue
		}
		if spec.interpolation == "${" && m.at("${", j) {
			j = m.codeEnd(j+2, '{', '}')
			continue
		}
		if m.at(close, j) {
			return literal{start: k, end: j + len(close), inner: [2]int{innerStart, j}}
		}
		j++
	}
	return literal{start: k, end: n, inner: [2]int{innerStart, n}}
}

func (m masker) regexAllowedBefore(k int) bool {
	j := k - 1
	for j >= 0 && (m.text[j] == ' ' || m.text[j] == '\t') {
		j--
	}
	if j < 0 || strings.IndexByte("(,=:[!&|?{};+-*%<>~^\n", m.text[j]) >= 0 {
		return true
	}
	return regexAfterWord.MatchString(m.text[unitsBack(m.text, j+1, 10) : j+1])
}

func (m masker) regexAt(k int) (literal, bool) {
	inClass := false
	for j := k + 1; j < len(m.text) && m.text[j] != '\n'; j++ {
		switch c := m.text[j]; {
		case c == '\\':
			j++
		case c == '[':
			inClass = true
		case c == ']':
			inClass = false
		case c == '/' && !inClass:
			return literal{start: k, end: j + 1, inner: [2]int{k + 1, j}}, true
		}
	}
	return literal{}, false
}
