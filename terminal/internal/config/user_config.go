// Package config is the terminal app's part of lazyreview's config, as the
// lazyreview command hands it over: read from config.yml, checked and
// resolved there, so the terminal app reads no file and no variable of it.
package config

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

// UserConfig is the terminal app's part of the config: the config file's
// path, the data home it reads reviews from, the editor a comment is written in, as a command line the shell
// runs with the file after it, the theme's colours, how the diff numbers
// its lines, and the keys.
type UserConfig struct {
	Path        string           `json:"path"`
	DataHome    string           `json:"dataHome"`
	Editor      string           `json:"editor"`
	Theme       ThemeConfig      `json:"theme"`
	LineNumbers string           `json:"lineNumbers"`
	Keybinding  KeybindingConfig `json:"keybinding"`
}

// Decode reads the hand-over, refusing a field it does not know and
// anything after the one value, which only a mistake in lazyreview causes.
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
