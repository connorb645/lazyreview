//go:build ignore

// Writes the config's documentation from its schema: the generated parts of
// docs/Config.md, and schema/config.json. go generate ./internal/config
// runs it, after a change to the schema.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/connorb645/lazyreview/terminal/internal/config"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

func main() {
	tr := i18n.EnglishTranslationSet()
	root := filepath.Join("..", "..", "..")
	docs := filepath.Join(root, "docs", "Config.md")
	text, err := os.ReadFile(docs)
	if err == nil {
		var written string
		if written, err = config.Documented(tr, string(text)); err == nil {
			err = os.WriteFile(docs, []byte(written), 0o644)
		}
	}
	if err == nil {
		schema := filepath.Join(root, "schema", "config.json")
		if err = os.MkdirAll(filepath.Dir(schema), 0o755); err == nil {
			err = os.WriteFile(schema, []byte(config.ConfigSchema(tr)), 0o644)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
