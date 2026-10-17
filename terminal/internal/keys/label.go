package keys

import "strings"

// Label is a key sequence as the help and the bar show it, such as "gg",
// "]p", "Space ?" or "Ctrl-o". Tab shows as Ctrl-i, the key a terminal
// sends for both, as Vim names its jump forward.
func Label(sequence string) string {
	parts := strings.Split(sequence, " ")
	var b strings.Builder
	for i, part := range parts {
		switch {
		case part == "space" && i < len(parts)-1:
			b.WriteString("Space ")
		case part == "tab":
			b.WriteString("Ctrl-i")
		case strings.HasPrefix(part, "ctrl+"):
			b.WriteString("Ctrl-" + strings.TrimPrefix(part, "ctrl+"))
		case len(part) > 1:
			b.WriteString(strings.ToUpper(part[:1]) + part[1:])
		default:
			b.WriteString(part)
		}
	}
	return b.String()
}
