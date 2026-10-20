package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

const standInSettings = "{\"accounts\": [\n  {\"login\": \"lee\"},\n  {\"login\": \"lee-at-work\", \"teams\": [{\"name\": \"example-org/mobile\", \"since\": \"2026-10-01\"}, {\"name\": \"example-org/web\"}]}\n]}\n"

type moveStandIn struct {
	home, data, settings, config string
}

func newMoveStandIn(t *testing.T, settings *string) moveStandIn {
	t.Helper()
	home := t.TempDir()
	s := moveStandIn{home: home, data: filepath.Join(home, ".lazyreview"), config: filepath.Join(home, "Library", "Application Support", "lazyreview", "config.yml")}
	s.settings = filepath.Join(s.data, "settings.json")
	if err := os.Mkdir(s.data, 0o755); err != nil {
		t.Fatal(err)
	}
	if settings != nil {
		writeText(t, s.settings, *settings)
	}
	return s
}

func writeText(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readText(t *testing.T, path string) string {
	t.Helper()
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func settingsText(text string) *string {
	return &text
}

// The check reads settings.json and the variables, shows what the move
// would write, writes nothing, and leaves out a variable equal to its
// default.
func TestTheMoveCheckShowsWhatItWouldWriteAndWritesNothing(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	s := newMoveStandIn(t, settingsText(standInSettings))
	plan, err := NewConfigMove(tr, nil).Plan(getenvOf(map[string]string{"LAZYREVIEW_MODEL": "claude-sonnet", "LAZYREVIEW_CLAUDE": "claude"}), "darwin", s.home)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"would write " + s.config + ":",
		"",
		"  # yaml-language-server: $schema=https://raw.githubusercontent.com/connorb645/lazyreview/main/schema/config.json",
		"  # lazyreview's config. docs/Config.md lists every setting and its default.",
		"  claude:",
		"    model: claude-sonnet",
		"  github:",
		"    accounts:",
		"      - login: lee",
		"      - login: lee-at-work",
		"        teams:",
		"          - name: example-org/mobile",
		"            since: '2026-10-01'",
		"          - name: example-org/web",
		"",
		"claude.model from LAZYREVIEW_MODEL",
		"github.accounts from " + s.settings,
		"would rename " + s.settings + " to settings.json.moved",
		"nothing was written; lazyreview config move makes the move",
	}
	if got := MoveCheckLines(tr, plan); !reflect.DeepEqual(got, want) {
		t.Errorf("the check says\n%s\nnot\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if readText(t, s.settings) != standInSettings || exists(filepath.Join(s.home, "Library")) {
		t.Error("the check wrote something")
	}
}

// The move loses nothing: every account, team and variable reads the same
// after it, with the variables or without them, and settings.json is kept
// byte for byte, with no temporary file left.
func TestTheMoveLosesNothingAndKeepsSettingsJSON(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	s := newMoveStandIn(t, settingsText(standInSettings))
	env := map[string]string{"LAZYREVIEW_HOME": s.data, "LAZYREVIEW_MODEL": "claude-sonnet", "LAZYREVIEW_EDITOR": "nvim"}
	move := NewConfigMove(tr, nil)
	plan, err := move.Plan(getenvOf(env), "darwin", s.home)
	if err != nil {
		t.Fatal(err)
	}
	said, err := move.Move(plan)
	if err != nil {
		t.Fatal(err)
	}
	if said[0] != "wrote "+s.config+":" {
		t.Errorf("the move starts by saying %q", said[0])
	}
	tail := []string{
		"",
		"renamed " + s.settings + " to settings.json.moved",
		"LAZYREVIEW_HOME is still set, and overrides dataHome while it is; taking it out of your shell's setup is safe, and not needed",
		"LAZYREVIEW_MODEL is still set, and overrides claude.model while it is; taking it out of your shell's setup is safe, and not needed",
		"LAZYREVIEW_EDITOR is still set, and overrides os.editor while it is; taking it out of your shell's setup is safe, and not needed",
	}
	if got := said[len(said)-len(tail):]; !reflect.DeepEqual(got, tail) {
		t.Errorf("the move ends saying\n%s", strings.Join(got, "\n"))
	}
	if readText(t, filepath.Join(s.data, "settings.json.moved")) != standInSettings || exists(s.settings) {
		t.Error("settings.json was not kept as settings.json.moved, byte for byte")
	}
	if got := names(t, filepath.Dir(s.config)); !reflect.DeepEqual(got, []string{"config.yml"}) {
		t.Errorf("the config folder holds %v", got)
	}
	since := "2026-10-01"
	accounts := []Account{{Login: "lee", Teams: []Team{}}, {Login: "lee-at-work", Teams: []Team{{Name: "example-org/mobile", Since: &since}, {Name: "example-org/web"}}}}
	withVariables, err := ReadUserConfig(tr, getenvOf(env), "darwin", s.home)
	if err != nil {
		t.Fatal(err)
	}
	fromFile, err := ReadUserConfig(tr, getenvOf(nil), "darwin", s.home)
	if err != nil {
		t.Fatal(err)
	}
	for _, read := range []Read{withVariables, fromFile} {
		v := read.Values
		claude, osPart, github := v["claude"].(map[string]any), v["os"].(map[string]any), v["github"].(map[string]any)
		if v["dataHome"] != s.data || claude["command"] != "claude" || claude["model"] != "claude-sonnet" || osPart["editor"] != "nvim" ||
			!reflect.DeepEqual(github["accounts"], accounts) || !reflect.DeepEqual(github["hiddenRepositories"], []string{}) {
			t.Errorf("the config reads as %v", v)
		}
	}
	checked := CheckedText(tr, fromFile)
	for _, value := range []string{"lee", "lee-at-work", "example-org/mobile", "2026-10-01", "example-org/web", "claude-sonnet", "nvim", s.data} {
		if !strings.Contains(checked, value) {
			t.Errorf("config check leaves out %s", value)
		}
	}
	if !regexp.MustCompile(`model: claude-sonnet +# LAZYREVIEW_MODEL\n`).MatchString(CheckedText(tr, withVariables)) ||
		!regexp.MustCompile(`model: claude-sonnet +# the file\n`).MatchString(checked) {
		t.Error("config check does not say where the model came from")
	}
}

// The move never overwrites a config, nor a settings.json.moved, and with
// nothing to move it writes nothing.
func TestTheMoveNeverOverwritesAndWithNothingToMoveWritesNothing(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	move := NewConfigMove(tr, nil)
	s := newMoveStandIn(t, settingsText(standInSettings))
	if err := os.MkdirAll(filepath.Dir(s.config), 0o755); err != nil {
		t.Fatal(err)
	}
	writeText(t, s.config, "server:\n  port: 4750\n")
	plan, err := move.Plan(getenvOf(nil), "darwin", s.home)
	if err != nil {
		t.Fatal(err)
	}
	refused := s.config + " exists already, and the move never overwrites a config"
	if lines := MoveCheckLines(tr, plan); lines[len(lines)-2] != refused {
		t.Errorf("the check says %q", lines[len(lines)-2])
	}
	if _, err := move.Move(plan); err == nil || err.Error() != refused {
		t.Errorf("the move gave %v", err)
	}
	if readText(t, s.config) != "server:\n  port: 4750\n" {
		t.Error("the config was overwritten")
	}
	if err := os.Remove(s.config); err != nil {
		t.Fatal(err)
	}
	writeText(t, filepath.Join(s.data, "settings.json.moved"), "kept")
	plan, err = move.Plan(getenvOf(nil), "darwin", s.home)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := move.Move(plan); err == nil || !strings.HasSuffix(err.Error(), "settings.json.moved exists already, so settings.json cannot be renamed to it; nothing was written") {
		t.Errorf("the move gave %v", err)
	}
	if readText(t, s.settings) != standInSettings {
		t.Error("settings.json changed")
	}

	none := newMoveStandIn(t, nil)
	plan, err = move.Plan(getenvOf(nil), "darwin", none.home)
	if err != nil {
		t.Fatal(err)
	}
	said := []string{"there is nothing to move: there is no " + none.settings + ", and no LAZYREVIEW_ variable is set"}
	moved, err := move.Move(plan)
	if !reflect.DeepEqual(MoveCheckLines(tr, plan), said) || err != nil || !reflect.DeepEqual(moved, said) || exists(none.config) {
		t.Errorf("with nothing to move, the move said %v, %v", moved, err)
	}
}

type failingRename struct {
	diskMoveFiles
	renames int
}

func (f *failingRename) Rename(from, to string) error {
	f.renames++
	if f.renames == 2 {
		return errors.New("the disk is full")
	}
	return f.diskMoveFiles.Rename(from, to)
}

// A move that fails partway puts back what it did, and a settings.json
// that is wrong says so, naming it.
func TestAMoveThatFailsPartwayPutsBackWhatItDid(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	s := newMoveStandIn(t, settingsText(standInSettings))
	move := NewConfigMove(tr, &failingRename{})
	plan, err := move.Plan(getenvOf(nil), "darwin", s.home)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := move.Move(plan); err == nil || err.Error() != "the move failed, and nothing changed: the disk is full" {
		t.Errorf("the move gave %v", err)
	}
	if readText(t, s.settings) != standInSettings || exists(filepath.Join(s.data, "settings.json.moved")) {
		t.Error("settings.json was not put back")
	}
	if got := names(t, filepath.Dir(s.config)); len(got) != 0 {
		t.Errorf("the config folder holds %v", got)
	}

	wrong := newMoveStandIn(t, settingsText(`{"accounts": [{"login": "lee author:alex"}]}`))
	if _, err := NewConfigMove(tr, nil).Plan(getenvOf(nil), "darwin", wrong.home); err == nil || err.Error() != wrong.settings+" gives account 1 no GitHub login" {
		t.Errorf("a wrong settings.json gave %v", err)
	}
}

// config check and config path say which file was read, and whether it
// exists, and name each setting the file gives that does nothing now.
func TestConfigCheckAndPathSayWhichFileWasRead(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	if got := PathLines(tr, ConfigFile{Path: "/x/config.yml"}); !reflect.DeepEqual(got, []string{"/x/config.yml", "there is no file there yet, so every setting has its default"}) {
		t.Errorf("config path says %v", got)
	}
	if got := PathLines(tr, ConfigFile{Path: "/x/config.yml", Exists: true}); !reflect.DeepEqual(got, []string{"/x/config.yml", "it exists"}) {
		t.Errorf("config path says %v", got)
	}
	read, err := ConfigFrom(tr, "server:\n  port: 4748\nkeybinding:\n  universal:\n    web: gW\n", recordedPath, recordedHome)
	if err != nil {
		t.Fatal(err)
	}
	read.Exists = true
	lines := CheckLines(tr, read)
	want := []string{
		"",
		"# server does nothing now, so it can go: lazyreview runs no server now; lazyreview terminal opens the app",
		"# keybinding.universal.web does nothing now, so it can go: lazyreview has no web app; keybinding.reviewView.drawings opens a written review's drawings in the browser",
	}
	if lines[0] != "the config at "+recordedPath+", on top of the defaults:" || lines[1] != "" || lines[2] != strings.TrimSuffix(CheckedText(tr, read), "\n") ||
		!reflect.DeepEqual(lines[3:], want) {
		t.Errorf("config check says\n%s", strings.Join(lines, "\n"))
	}
	read.Exists, read.Unused = false, nil
	if lines := CheckLines(tr, read); len(lines) != 3 || lines[0] != "no config at "+recordedPath+", so every setting has its default:" {
		t.Errorf("config check with no file says\n%s", strings.Join(lines, "\n"))
	}
}
