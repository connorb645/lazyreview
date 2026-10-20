// Package languages reads the types a source file declares, in the
// languages the map draws: Swift, Kotlin, Java, TypeScript and JavaScript,
// and Python. Each reader takes a file's text and gives its declarations
// and imports, as lazyreview has always read them.
package languages

import "strings"

// Families is, per family name, the words that declare a type and the
// files that hold its code. Types resolve across the files of one family
// only, so Kotlin and Java share one.
var Families = map[string]Family{
	"swift":  {Keywords: []string{"class", "struct", "enum", "protocol", "actor"}, Globs: []string{"*.swift"}},
	"jvm":    {Keywords: []string{"class", "interface", "object", "enum", "record"}, Globs: []string{"*.kt", "*.kts", "*.java"}},
	"js":     {Keywords: []string{"class", "interface", "enum", "type"}, Globs: []string{"*.ts", "*.tsx", "*.mts", "*.cts", "*.js", "*.jsx", "*.mjs", "*.cjs"}},
	"python": {Keywords: []string{"class"}, Globs: []string{"*.py", "*.pyi"}},
}

var languages = []Language{
	{Name: "Swift", Family: "swift", Extensions: []string{".swift"}, Read: readSwift},
	{Name: "Kotlin", Family: "jvm", Extensions: []string{".kt", ".kts"}, Read: readKotlin},
	{Name: "Java", Family: "jvm", Extensions: []string{".java"}, Read: readJava},
	{Name: "TypeScript", Family: "js", Extensions: []string{".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs"}, Read: readTypeScript},
	{Name: "Python", Family: "python", Extensions: []string{".py", ".pyi"}, Read: readPython},
}

// LanguageFor is the language path is written in, chosen by its extension,
// or nil when the map cannot read its types.
func LanguageFor(path string) *Language {
	lower := strings.ToLower(path)
	for _, language := range languages {
		for _, extension := range language.Extensions {
			if strings.HasSuffix(lower, extension) {
				return &language
			}
		}
	}
	return nil
}
