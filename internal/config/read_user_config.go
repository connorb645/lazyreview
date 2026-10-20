package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/utils"
)

// ReadUserConfig reads lazyreview's config: the file FindConfigFile finds,
// on top of the defaults, and each variable over both, as lazyreview has
// always read it. A problem with the file names the file and every problem
// at once, each with what the setting must be.
func ReadUserConfig(tr *i18n.TranslationSet, getenv func(string) string, platform, home string) (Read, error) {
	file, err := FindConfigFile(tr, getenv, platform, home, IsFile)
	if err != nil {
		return Read{}, err
	}
	text := ""
	if file.Exists {
		raw, err := os.ReadFile(file.Path)
		if err != nil {
			return Read{}, errors.New(utils.ResolvePlaceholderString(tr.ConfigUnreadable, map[string]string{"path": file.Path, "why": err.Error()}))
		}
		text = string(raw)
	}
	read, err := ConfigFrom(tr, text, file.Path, home)
	if err != nil {
		return Read{}, err
	}
	read.Exists = file.Exists
	return read, overridden(tr, read, getenv)
}

// ConfigFrom is the config the text of the file at path gives, on top of
// the defaults, with each setting's source.
func ConfigFrom(tr *i18n.TranslationSet, text, path, home string) (Read, error) {
	given, err := ParseYAML(tr, text)
	if err != nil {
		return Read{}, errors.New(utils.ResolvePlaceholderString(tr.ConfigUnreadable, map[string]string{"path": path, "why": err.Error()}))
	}
	schema := Schema(tr)
	read := Read{Path: path, Values: Defaults(schema), Sources: map[string]string{}, Unused: []string{}}
	for _, leaf := range Leaves(schema, "") {
		read.Sources[leaf.Key] = "default"
	}
	if given != nil {
		top, isMap := given.(YAMLMap)
		if !isMap {
			return Read{}, errors.New(utils.ResolvePlaceholderString(tr.ConfigProblem, map[string]string{"path": path, "problem": tr.ConfigNotAMap}))
		}
		var problems []string
		merge(tr, schema, top, read.Values, read.Sources, "", &problems, &read.Unused)
		if len(problems) > 0 {
			return Read{}, refusedFor(tr, path, problems)
		}
	}
	data := read.Values["dataHome"].(string)
	switch {
	case data == "~":
		read.Values["dataHome"] = home
	case strings.HasPrefix(data, "~/"):
		read.Values["dataHome"] = filepath.Join(home, data[2:])
	}
	return read, nil
}

func refusedFor(tr *i18n.TranslationSet, path string, problems []string) error {
	if len(problems) == 1 {
		return errors.New(utils.ResolvePlaceholderString(tr.ConfigProblem, map[string]string{"path": path, "problem": problems[0]}))
	}
	listed := make([]string, len(problems))
	for i, p := range problems {
		listed[i] = utils.ResolvePlaceholderString(tr.ConfigListedProblem, map[string]string{"problem": p})
	}
	return errors.New(utils.ResolvePlaceholderString(tr.ConfigProblems, map[string]string{"path": path, "count": strconv.Itoa(len(problems)), "problems": strings.Join(listed, "\n")}))
}

func merge(tr *i18n.TranslationSet, section *Section, given YAMLMap, values map[string]any, sources map[string]string, prefix string, problems, unused *[]string) {
	for _, name := range given.Keys {
		value := given.Values[name]
		key := name
		if prefix != "" {
			key = prefix + "." + name
		}
		var entry *Entry
		for i := range section.Entries {
			if section.Entries[i].Name == name {
				entry = &section.Entries[i]
			}
		}
		if entry == nil {
			if _, ok := Unused(tr)[key]; ok {
				*unused = append(*unused, key)
			} else if why, ok := Retired(tr)[key]; ok {
				*problems = append(*problems, utils.ResolvePlaceholderString(tr.ConfigRetired, map[string]string{"key": key, "why": why}))
			} else {
				*problems = append(*problems, utils.ResolvePlaceholderString(tr.ConfigUnknown, map[string]string{"key": key}))
			}
			continue
		}
		if entry.Section != nil {
			inner, isMap := value.(YAMLMap)
			if !isMap {
				*problems = append(*problems, utils.ResolvePlaceholderString(tr.ConfigNotASection, map[string]string{"key": key, "value": Shown(value)}))
				continue
			}
			merge(tr, entry.Section, inner, values[name].(map[string]any), sources, key, problems, unused)
			continue
		}
		read, found := check(tr, entry.Setting, key, value)
		if len(found) > 0 {
			*problems = append(*problems, found...)
			continue
		}
		values[name] = read
		sources[key] = "file"
	}
}

// overridden puts each variable over the file and the defaults, as each
// has always read: an empty one is not set.
func overridden(tr *i18n.TranslationSet, read Read, getenv func(string) string) error {
	set := func(key, name string, apply func(string) error) error {
		given := getenv(name)
		if given == "" {
			return nil
		}
		read.Sources[key] = name
		return apply(given)
	}
	section := func(name string) map[string]any { return read.Values[name].(map[string]any) }
	return errors.Join(
		set("dataHome", "LAZYREVIEW_HOME", func(given string) error { read.Values["dataHome"] = absolutePath(given); return nil }),
		set("claude.command", "LAZYREVIEW_CLAUDE", func(given string) error { section("claude")["command"] = given; return nil }),
		set("claude.model", "LAZYREVIEW_MODEL", func(given string) error { section("claude")["model"] = given; return nil }),
		set("os.editor", "LAZYREVIEW_EDITOR", func(given string) error { section("os")["editor"] = given; return nil }),
	)
}
