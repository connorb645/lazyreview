package config

import "strings"

// jsonText is a value as JavaScript's JSON.stringify writes it two spaces
// deep: a YAMLMap as an object, its keys in their order.
func jsonText(v any, indent string) string {
	inner := indent + "  "
	switch x := v.(type) {
	case YAMLMap:
		if len(x.Keys) == 0 {
			return "{}"
		}
		parts := make([]string, len(x.Keys))
		for i, k := range x.Keys {
			parts[i] = inner + jsString(k) + ": " + jsonText(x.Values[k], inner)
		}
		return "{\n" + strings.Join(parts, ",\n") + "\n" + indent + "}"
	case []any:
		if len(x) == 0 {
			return "[]"
		}
		parts := make([]string, len(x))
		for i, item := range x {
			parts[i] = inner + jsonText(item, inner)
		}
		return "[\n" + strings.Join(parts, ",\n") + "\n" + indent + "]"
	case []string:
		items := make([]any, len(x))
		for i, s := range x {
			items[i] = s
		}
		return jsonText(items, indent)
	case []Account:
		return "[]"
	}
	return Shown(v)
}
