package config

import (
	"errors"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// ConfigSchema is the JSON schema editors read for config.yml, as
// schema/config.json holds it, with its final new line.
func ConfigSchema(tr *i18n.TranslationSet) string {
	schema := joinedMap(ordered("$schema", "http://json-schema.org/draft-07/schema#", "title", tr.ConfigSchemaTitle, "description", tr.ConfigSchemaDescription), sectionSchema(Schema(tr)))
	return jsonText(schema, "") + "\n"
}

// Documented is docs/Config.md, markdown, with its generated parts written
// again from the schema: the table of the variables that override a
// setting, and the default config. Each part sits between its markers,
// which stay as they are.
func Documented(tr *i18n.TranslationSet, markdown string) (string, error) {
	var variables []string
	variables = append(variables, "| "+tr.ConfigDocsVariable+" | "+tr.ConfigDocsOverrides+" |", "| --- | --- |")
	for _, leaf := range Leaves(Schema(tr), "") {
		if leaf.Setting.Env != "" {
			variables = append(variables, "| `"+leaf.Setting.Env+"` | `"+leaf.Key+"` |")
		}
	}
	defaults := append(append([]string{"```yaml"}, strings.Split(strings.TrimRight(DefaultConfigText(tr), "\n"), "\n")...), "```")
	for _, part := range []struct {
		name  string
		lines []string
	}{{"VARIABLES", variables}, {"DEFAULT CONFIG", defaults}} {
		start := "<!-- START " + part.name + ": " + tr.ConfigDocsGenerated + " -->"
		end := "<!-- END " + part.name + " -->"
		from, to := strings.Index(markdown, start), strings.Index(markdown, end)
		if from < 0 || to < from {
			return "", errors.New(utils.ResolvePlaceholderString(tr.ConfigDocsNoMarkers, map[string]string{"name": part.name}))
		}
		markdown = markdown[:from+len(start)] + "\n" + strings.Join(part.lines, "\n") + "\n" + markdown[to:]
	}
	return markdown, nil
}

func sectionSchema(s *Section) YAMLMap {
	properties := YAMLMap{Values: map[string]any{}}
	for _, e := range s.Entries {
		properties.Keys = append(properties.Keys, e.Name)
		if e.Section != nil {
			properties.Values[e.Name] = sectionSchema(e.Section)
		} else {
			properties.Values[e.Name] = settingSchema(e.Setting)
		}
	}
	out := ordered("type", "object")
	if len(s.About) > 0 {
		out = joinedMap(out, ordered("description", strings.Join(s.About, " ")))
	}
	return joinedMap(out, ordered("properties", properties, "additionalProperties", false))
}

func settingSchema(s *Setting) YAMLMap {
	out := typeSchema(s.Check)
	if s.Choices != nil {
		out = joinedMap(out, ordered("enum", s.Choices))
	}
	if s.Check == "numberedKeys" {
		count := float64(len(s.Default.([]string)))
		out = joinedMap(out, ordered("minItems", count, "maxItems", count))
	}
	if len(s.About) > 0 {
		out = joinedMap(out, ordered("description", strings.Join(s.About, " ")))
	}
	return joinedMap(out, ordered("default", s.Default))
}

func typeSchema(check string) YAMLMap {
	colour := []any{ordered("type", "string", "pattern", Patterns.Colour), ordered("type", "integer", "minimum", 0.0, "maximum", 255.0)}
	switch check {
	case "folder":
		return ordered("type", "string", "pattern", "^(~|~/.*|/.*)$")
	case "port":
		return ordered("type", "integer", "minimum", 1.0, "maximum", 65535.0)
	case "command":
		return ordered("type", "string", "minLength", 1.0)
	case "text", "choice":
		return ordered("type", "string")
	case "colour":
		return ordered("oneOf", colour)
	case "keys":
		return ordered("oneOf", []any{ordered("type", "string", "minLength", 1.0), ordered("type", "array", "minItems", 1.0, "items", ordered("type", "string", "minLength", 1.0))})
	case "numberedKeys":
		return ordered("type", "array", "items", ordered("type", "string", "minLength", 1.0))
	case "colours":
		return ordered("type", "array", "minItems", 8.0, "maxItems", 8.0, "items", ordered("oneOf", colour))
	case "repositories":
		return ordered("type", []any{"array", "null"}, "items", ordered("type", "string", "pattern", Patterns.Repository))
	case "accounts":
		team := ordered("type", "object", "required", []any{"name"},
			"properties", ordered("name", ordered("type", "string", "pattern", Patterns.Team), "since", ordered("type", "string", "pattern", Patterns.Date)),
			"additionalProperties", false)
		account := ordered("type", "object", "required", []any{"login"},
			"properties", ordered("login", ordered("type", "string", "pattern", Patterns.Login), "teams", ordered("type", "array", "items", team)),
			"additionalProperties", false)
		return ordered("type", "array", "items", account)
	}
	return YAMLMap{Values: map[string]any{}}
}
