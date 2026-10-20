package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const movedSettings = "settings.json.moved"

var movedVariables = []namedValue{
	{name: "LAZYREVIEW_HOME", value: "dataHome"},
	{name: "LAZYREVIEW_CLAUDE", value: "claude.command"},
	{name: "LAZYREVIEW_MODEL", value: "claude.model"},
	{name: "LAZYREVIEW_EDITOR", value: "os.editor"},
}

// ConfigMove is the one-time move of settings.json and the LAZYREVIEW_
// variables into the config file.
type ConfigMove struct {
	tr    *i18n.TranslationSet
	files MoveFiles
}

// NewConfigMove makes the move, reading and writing through files, or the
// disk when files is nil.
func NewConfigMove(tr *i18n.TranslationSet, files MoveFiles) *ConfigMove {
	if files == nil {
		files = diskMoveFiles{}
	}
	return &ConfigMove{tr: tr, files: files}
}

// Plan is what the move would do, reading and writing nothing but
// settings.json: the accounts it holds, in the data home, and each of
// lazyreview's own variables that is set. It fails when settings.json
// cannot be read, naming it and what is wrong.
func (m *ConfigMove) Plan(getenv func(string) string, platform, home string) (MovePlan, error) {
	file, err := FindConfigFile(m.tr, getenv, platform, home, m.files.Exists)
	if err != nil {
		return MovePlan{}, err
	}
	dataHome := filepath.Join(home, ".lazyreview")
	if given := getenv("LAZYREVIEW_HOME"); given != "" {
		dataHome = absolutePath(given)
	}
	settings := filepath.Join(dataHome, "settings.json")
	plan := MovePlan{ConfigPath: file.Path, ConfigExists: file.Exists, SettingsPath: settings, SettingsExists: m.files.Exists(settings),
		Values: YAMLMap{Values: map[string]any{}}, From: []MoveSource{}, Variables: []string{}}
	defaults := Defaults(Schema(m.tr))
	give := func(key string, value any, source string) {
		section, name, inner := strings.Cut(key, ".")
		was := defaults[section]
		if inner {
			was = defaults[section].(map[string]any)[name]
		}
		if accounts, ok := was.([]Account); ok {
			was = accountValues(accounts)
		}
		if Shown(value) == Shown(was) {
			return
		}
		if inner {
			held, _ := plan.Values.Values[section].(YAMLMap)
			put(&held, name, value)
			value = held
		}
		put(&plan.Values, section, value)
		plan.From = append(plan.From, MoveSource{Key: key, Source: source})
	}
	for _, v := range movedVariables {
		given := getenv(v.name)
		if given == "" {
			continue
		}
		if v.value == "dataHome" {
			given = dataHome
		}
		give(v.value, given, v.name)
		plan.Variables = append(plan.Variables, v.name)
	}
	if plan.SettingsExists {
		accounts, err := m.settingsAccounts(settings)
		if err != nil {
			return MovePlan{}, err
		}
		give("github.accounts", accounts, settings)
	}
	return plan, nil
}

// MoveCheckLines is what lazyreview config move --check prints: the file
// the move would write, where each value comes from, the rename it would
// make, and that the move would refuse when a config file exists.
func MoveCheckLines(tr *i18n.TranslationSet, plan MovePlan) []string {
	if nothingToMove(plan) {
		return []string{utils.ResolvePlaceholderString(tr.ConfigMoveNothing, map[string]string{"settings": plan.SettingsPath})}
	}
	out := append(append([]string{utils.ResolvePlaceholderString(tr.ConfigMoveWouldWrite, map[string]string{"path": plan.ConfigPath}), ""},
		indentedText(MovedText(tr, plan.Values))...), "")
	for _, from := range plan.From {
		out = append(out, utils.ResolvePlaceholderString(tr.ConfigMoveFrom, map[string]string{"key": from.Key, "source": from.Source}))
	}
	if plan.SettingsExists {
		out = append(out, utils.ResolvePlaceholderString(tr.ConfigMoveWouldRename, map[string]string{"path": plan.SettingsPath, "name": movedSettings}))
	}
	if plan.ConfigExists {
		out = append(out, utils.ResolvePlaceholderString(tr.ConfigMoveRefused, map[string]string{"path": plan.ConfigPath}))
	}
	return append(out, tr.ConfigMoveNothingWritten)
}

// Move makes the move: it refuses when a config file exists, writes the
// file under a temporary name, renames settings.json to
// settings.json.moved, keeping its bytes, then gives the file its name.
// When a step fails it puts back what was done, and says what it could
// not. It returns the lines saying what it did, naming each variable still
// set, which overrides the config while it is.
func (m *ConfigMove) Move(plan MovePlan) ([]string, error) {
	if nothingToMove(plan) {
		return []string{utils.ResolvePlaceholderString(m.tr.ConfigMoveNothing, map[string]string{"settings": plan.SettingsPath})}, nil
	}
	if plan.ConfigExists || m.files.Exists(plan.ConfigPath) {
		return nil, errors.New(utils.ResolvePlaceholderString(m.tr.ConfigMoveRefused, map[string]string{"path": plan.ConfigPath}))
	}
	moved := filepath.Join(filepath.Dir(plan.SettingsPath), movedSettings)
	if plan.SettingsExists && m.files.Exists(moved) {
		return nil, errors.New(utils.ResolvePlaceholderString(m.tr.ConfigMoveMovedExists, map[string]string{"path": moved}))
	}
	text := MovedText(m.tr, plan.Values)
	temporary := filepath.Join(filepath.Dir(plan.ConfigPath), "."+filepath.Base(plan.ConfigPath)+"."+strconv.Itoa(os.Getpid())+".moving")
	if err := m.steps(plan, temporary, moved, text); err != nil {
		return nil, err
	}
	out := append(append([]string{utils.ResolvePlaceholderString(m.tr.ConfigMoveWrote, map[string]string{"path": plan.ConfigPath}), ""}, indentedText(text)...), "")
	if plan.SettingsExists {
		out = append(out, utils.ResolvePlaceholderString(m.tr.ConfigMoveRenamed, map[string]string{"path": plan.SettingsPath, "name": movedSettings}))
	}
	for _, name := range plan.Variables {
		for _, v := range movedVariables {
			if v.name == name {
				out = append(out, utils.ResolvePlaceholderString(m.tr.ConfigMoveStillSet, map[string]string{"name": name, "key": v.value}))
			}
		}
	}
	return out, nil
}

func (m *ConfigMove) steps(plan MovePlan, temporary, moved, text string) error {
	var undo []func() error
	failed := func(err error) error {
		var stuck []string
		for i := len(undo) - 1; i >= 0; i-- {
			if err := undo[i](); err != nil {
				stuck = append(stuck, err.Error())
			}
		}
		if len(stuck) > 0 {
			return errors.New(utils.ResolvePlaceholderString(m.tr.ConfigMoveFailedStuck, map[string]string{"why": err.Error(), "stuck": strings.Join(stuck, "; ")}))
		}
		return errors.New(utils.ResolvePlaceholderString(m.tr.ConfigMoveFailed, map[string]string{"why": err.Error()}))
	}
	if err := m.files.Folder(filepath.Dir(plan.ConfigPath)); err != nil {
		return failed(err)
	}
	if err := m.files.Write(temporary, text); err != nil {
		return failed(err)
	}
	undo = append(undo, func() error { return m.files.Remove(temporary) })
	if plan.SettingsExists {
		if err := m.files.Rename(plan.SettingsPath, moved); err != nil {
			return failed(err)
		}
		undo = append(undo, func() error { return m.files.Rename(moved, plan.SettingsPath) })
	}
	if err := m.files.Rename(temporary, plan.ConfigPath); err != nil {
		return failed(err)
	}
	return nil
}

func (m *ConfigMove) settingsAccounts(path string) ([]any, error) {
	unreadable := func(err error) error {
		return errors.New(utils.ResolvePlaceholderString(m.tr.ConfigUnreadable, map[string]string{"path": path, "why": err.Error()}))
	}
	text, err := m.files.Read(path)
	if err != nil {
		return nil, unreadable(err)
	}
	given, err := orderedJSON(text)
	if err != nil {
		return nil, unreadable(err)
	}
	settings, isMap := given.(YAMLMap)
	listed, isList := settings.Values["accounts"].([]any)
	if !isMap || !isList {
		return nil, refusedFor(m.tr, path, []string{m.tr.ConfigMoveNeedsAccounts})
	}
	checked, problems := checkAccounts(m.tr, listed, "accounts")
	if len(problems) > 0 {
		return nil, refusedFor(m.tr, path, problems)
	}
	accounts := checked.([]Account)
	out := make([]any, len(accounts))
	for i, a := range accounts {
		if len(a.Teams) == 0 {
			out[i] = ordered("login", a.Login)
			continue
		}
		teams := make([]any, len(a.Teams))
		for j, t := range a.Teams {
			teams[j] = ordered("name", t.Name)
			if t.Since != nil {
				teams[j] = ordered("name", t.Name, "since", *t.Since)
			}
		}
		out[i] = ordered("login", a.Login, "teams", teams)
	}
	return out, nil
}

func put(m *YAMLMap, key string, value any) {
	if m.Values == nil {
		m.Values = map[string]any{}
	}
	if _, held := m.Values[key]; !held {
		m.Keys = append(m.Keys, key)
	}
	m.Values[key] = value
}

func nothingToMove(plan MovePlan) bool {
	return !plan.SettingsExists && len(plan.Variables) == 0
}

func indentedText(text string) []string {
	lines := strings.Split(jsTrimmed(text), "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return lines
}
