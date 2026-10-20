package config

import (
	"strings"

	"github.com/connorb645/lazyreview/internal/i18n"
)

const schemaAddress = "https://raw.githubusercontent.com/connorb645/lazyreview/main/schema/config.json"

// MovedText is the config file the move writes: the line that gives an
// editor the schema, the line that names the file, then the values.
func MovedText(tr *i18n.TranslationSet, values YAMLMap) string {
	return strings.Join([]string{"# yaml-language-server: $schema=" + schemaAddress, "# " + tr.ConfigMovedHeader, jsTrimmed(dumpYAML(values))}, "\n") + "\n"
}
