package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
)

// docs/Config.md and schema/config.json say what the schema says; go
// generate ./internal/config writes them again.
func TestTheConfigDocsAreWrittenFromTheSchema(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	root := filepath.Join("..", "..")
	docs, err := os.ReadFile(filepath.Join(root, "docs", "Config.md"))
	if err != nil {
		t.Fatal(err)
	}
	written, err := Documented(tr, string(docs))
	if err != nil || written != string(docs) {
		t.Errorf("docs/Config.md is out of date with the schema, %v; run go generate ./internal/config", err)
	}
	schema, err := os.ReadFile(filepath.Join(root, "schema", "config.json"))
	if err != nil || string(schema) != ConfigSchema(tr) {
		t.Errorf("schema/config.json is out of date with the schema, %v; run go generate ./internal/config", err)
	}
}

// The default config, as the docs show it, reads back as the defaults.
func TestTheDefaultConfigReadsBackAsTheDefaults(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	read, err := ConfigFrom(tr, DefaultConfigText(tr), "config.yml", "/home/lee")
	if err != nil {
		t.Fatal(err)
	}
	want := Defaults(Schema(tr))
	want["dataHome"] = "/home/lee/.lazyreview"
	if got := plain(t, read.Values); !equalJSON(t, got, plain(t, want)) {
		t.Errorf("the default config reads as %v", got)
	}
}
