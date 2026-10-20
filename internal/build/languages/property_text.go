package languages

import (
	"strings"
	"unicode/utf8"
)

func shownType(text, masked string, from, to int) string {
	for from < to {
		r, size := utf8.DecodeRuneInString(masked[from:to])
		if !isJSSpace(r) {
			break
		}
		from += size
	}
	for to > from {
		r, size := utf8.DecodeLastRuneInString(masked[from:to])
		if !isJSSpace(r) {
			break
		}
		to -= size
	}
	return unitBlanked(text, masked, from, to)
}

func unitBlanked(text, masked string, from, to int) string {
	var out strings.Builder
	for i := from; i < to; {
		r, size := utf8.DecodeRuneInString(text[i:to])
		if size > 1 && masked[i] == ' ' {
			out.WriteString(strings.Repeat(" ", unitsOf(r)))
		} else {
			out.WriteString(masked[i : i+size])
		}
		i += size
	}
	return out.String()
}

func storedProperty(name, written, inferred string, line int) Property {
	p := Property{Name: name, Line: line}
	switch {
	case written != "":
		p.Type = &written
	case inferred != "":
		p.Type = &inferred
		p.Inferred = true
	}
	return p
}
