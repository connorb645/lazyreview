// Package config is lazyreview's config: config.yml, found as lazygit finds
// its own, read on top of the defaults its schema gives and checked
// setting by setting, the variables that override it, and the
// documentation made from the schema.
package config

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/connorb645/lazyreview/internal/i18n"
)

// UserConfig is the terminal app's part of the config: the config file's
// path, the data home it reads reviews from, the editor a comment is
// written in, as a command line the shell runs with the file after it, the
// Claude Code it asks, the theme's colours, how the diff numbers its lines,
// and the keys.
type UserConfig struct {
	Path        string           `json:"path"`
	DataHome    string           `json:"dataHome"`
	Editor      string           `json:"editor"`
	Claude      ClaudeConfig     `json:"claude"`
	Theme       ThemeConfig      `json:"theme"`
	LineNumbers string           `json:"lineNumbers"`
	Keybinding  KeybindingConfig `json:"keybinding"`
}

// Decode reads the terminal app's part of the config written as JSON,
// refusing a field it does not know and anything after the one value.
func Decode(text string, tr *i18n.TranslationSet) (UserConfig, error) {
	var c UserConfig
	decoder := json.NewDecoder(bytes.NewReader([]byte(text)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		return UserConfig{}, err
	}
	if decoder.More() {
		return UserConfig{}, errors.New(tr.MoreThanOneValue)
	}
	return c, nil
}
