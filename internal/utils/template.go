package utils

import "strings"

// ResolvePlaceholderString fills a string's named placeholders, written
// {{.name}} or {{name}}, with the values the arguments give each name.
func ResolvePlaceholderString(str string, arguments map[string]string) string {
	oldnews := make([]string, 0, len(arguments)*4)
	for key, value := range arguments {
		oldnews = append(oldnews,
			"{{"+key+"}}", value,
			"{{."+key+"}}", value,
		)
	}
	return strings.NewReplacer(oldnews...).Replace(str)
}
