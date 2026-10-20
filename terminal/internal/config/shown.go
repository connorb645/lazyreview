package config

import (
	"math"
	"strconv"
	"strings"
)

// Shown is a value as the config's words show it: as JSON, the way
// lazyreview has always written it, a mapping's keys in their order.
func Shown(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return strconv.FormatBool(x)
	case float64:
		return jsNumber(x)
	case int:
		return strconv.Itoa(x)
	case string:
		return jsString(x)
	case []any:
		parts := make([]string, len(x))
		for i, item := range x {
			parts[i] = Shown(item)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case []string:
		parts := make([]string, len(x))
		for i, item := range x {
			parts[i] = jsString(item)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case YAMLMap:
		parts := make([]string, len(x.Keys))
		for i, k := range x.Keys {
			parts[i] = jsString(k) + ":" + Shown(x.Values[k])
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	return "null"
}

// jsNumber is a number as JavaScript writes it: no point for a whole
// number below 1e21, and an exponent only outside 1e-6 to 1e21.
func jsNumber(f float64) string {
	switch {
	case math.IsNaN(f) || math.IsInf(f, 0):
		return "null"
	case f == 0:
		return "0"
	}
	if abs := math.Abs(f); abs >= 1e-6 && abs < 1e21 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	text := strconv.FormatFloat(f, 'e', -1, 64)
	mantissa, exponent, _ := strings.Cut(text, "e")
	sign := exponent[:1]
	digits := strings.TrimLeft(exponent[1:], "0")
	return mantissa + "e" + sign + digits
}

// jsString is a value as JavaScript's String writes it, and a string as
// JSON writes it, quoted.
func jsString(v any) string {
	switch x := v.(type) {
	case string:
		var out strings.Builder
		out.WriteByte('"')
		for _, r := range x {
			switch r {
			case '"':
				out.WriteString(`\"`)
			case '\\':
				out.WriteString(`\\`)
			case '\n':
				out.WriteString(`\n`)
			case '\r':
				out.WriteString(`\r`)
			case '\t':
				out.WriteString(`\t`)
			case '\b':
				out.WriteString(`\b`)
			case '\f':
				out.WriteString(`\f`)
			default:
				if r < 0x20 {
					out.WriteString(`\u00`)
					out.WriteString(strconv.FormatInt(int64(r)>>4, 16))
					out.WriteString(strconv.FormatInt(int64(r)&0xf, 16))
				} else {
					out.WriteRune(r)
				}
			}
		}
		out.WriteByte('"')
		return out.String()
	}
	return Shown(v)
}

// jsText is a value as JavaScript's String writes it.
func jsText(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return x
	case float64:
		if math.IsNaN(x) {
			return "NaN"
		}
		if math.IsInf(x, 0) {
			if x > 0 {
				return "Infinity"
			}
			return "-Infinity"
		}
		return jsNumber(x)
	}
	return Shown(v)
}
