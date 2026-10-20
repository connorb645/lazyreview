package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// StableJSON is v as JSON with every object's keys in sorted order and each
// text written as JavaScript's JSON.stringify writes it, so a hash of it is
// the same hash every earlier lazyreview gave the same value.
func StableJSON(v any) (string, error) {
	text, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	decoder := json.NewDecoder(bytes.NewReader(text))
	decoder.UseNumber()
	var plain any
	if err := decoder.Decode(&plain); err != nil {
		return "", err
	}
	var out strings.Builder
	writeStable(&out, plain)
	return out.String(), nil
}

func writeStable(out *strings.Builder, v any) {
	switch x := v.(type) {
	case []any:
		out.WriteByte('[')
		for i, item := range x {
			if i > 0 {
				out.WriteByte(',')
			}
			writeStable(out, item)
		}
		out.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				out.WriteByte(',')
			}
			out.WriteString(jsString(k))
			out.WriteByte(':')
			writeStable(out, x[k])
		}
		out.WriteByte('}')
	case string:
		out.WriteString(jsString(x))
	case json.Number:
		out.WriteString(x.String())
	case bool:
		fmt.Fprint(out, x)
	case nil:
		out.WriteString("null")
	}
}

func jsString(s string) string {
	var out strings.Builder
	out.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			out.WriteString(`\"`)
		case '\\':
			out.WriteString(`\\`)
		case '\b':
			out.WriteString(`\b`)
		case '\f':
			out.WriteString(`\f`)
		case '\n':
			out.WriteString(`\n`)
		case '\r':
			out.WriteString(`\r`)
		case '\t':
			out.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&out, `\u%04x`, r)
			} else {
				out.WriteRune(r)
			}
		}
	}
	out.WriteByte('"')
	return out.String()
}
