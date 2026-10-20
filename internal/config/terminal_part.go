package config

import (
	"encoding/json"

	"github.com/connorb645/lazyreview/internal/i18n"
)

// TerminalPart is the terminal app's part of the config read: the file's
// path, the data home, the editor, chosen from the config's os.editor, then
// VISUAL, then EDITOR, then vi, the Claude Code it asks, the theme's
// colours, how the diff numbers its lines, and the keys of every place,
// each action's as a list of its key sequences as the terminal app's key
// reader names them; a numbered list keeps its places, with "" for a place
// that has no key.
func TerminalPart(tr *i18n.TranslationSet, read Read, getenv func(string) string) (UserConfig, error) {
	editor := ""
	for _, given := range []string{section(read.Values, "os")["editor"].(string), getenv("VISUAL"), getenv("EDITOR"), "vi"} {
		if editor == "" {
			editor = given
		}
	}
	gui := section(read.Values, "gui")
	part := map[string]any{
		"path": read.Path, "dataHome": read.Values["dataHome"], "editor": editor, "claude": read.Values["claude"],
		"theme": gui["theme"], "lineNumbers": gui["lineNumbers"], "keybinding": readerNames(Schema(tr), section(read.Values, "keybinding")),
	}
	text, err := json.Marshal(part)
	if err != nil {
		return UserConfig{}, err
	}
	return Decode(string(text), tr)
}

func section(values map[string]any, name string) map[string]any {
	return values[name].(map[string]any)
}

func readerNames(schema *Section, keybinding map[string]any) map[string]map[string][]string {
	var places *Section
	for _, e := range schema.Entries {
		if e.Name == "keybinding" {
			places = e.Section
		}
	}
	out := map[string]map[string][]string{}
	for _, place := range places.Entries {
		actions := keybinding[place.Name].(map[string]any)
		out[place.Name] = map[string][]string{}
		for _, action := range place.Section.Entries {
			labels, _ := actions[action.Name].([]string)
			if one, isOne := actions[action.Name].(string); isOne {
				labels = []string{one}
			}
			names := []string{}
			for _, label := range labels {
				if action.Setting.Check == "numberedKeys" || label != Disabled {
					names = append(names, ReaderName(label))
				}
			}
			out[place.Name][action.Name] = names
		}
	}
	return out
}
