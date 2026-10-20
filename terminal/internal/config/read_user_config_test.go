package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

func TestWithNoConfigEverySettingHasItsDefaultTheDataHomeInTheHomeFolder(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	dir := t.TempDir()
	read, err := ReadUserConfig(tr, env(map[string]string{"LAZYREVIEW_CONFIG_DIR": dir}), "darwin", "/home/lee")
	if err != nil || read.Path != filepath.Join(dir, "config.yml") || read.Exists || read.Values["dataHome"] != "/home/lee/.lazyreview" {
		t.Fatalf("with no config, the read is %+v, %v", read, err)
	}
	for key, source := range read.Sources {
		if source != "default" {
			t.Errorf("%s comes from %s", key, source)
		}
	}
}

func TestEachVariableOverridesItsSettingOverTheFileAndSaysSo(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte("server:\n  port: 4748\nos:\n  editor: nano\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	vars := map[string]string{"LAZYREVIEW_CONFIG_DIR": dir, "LAZYREVIEW_HOME": "/srv/lr", "LAZYREVIEW_PORT": "4749", "LAZYREVIEW_CLAUDE": "/opt/claude", "LAZYREVIEW_MODEL": "claude-opus", "LAZYREVIEW_EDITOR": "nvim"}
	read, err := ReadUserConfig(tr, env(vars), "darwin", "/home/lee")
	if err != nil {
		t.Fatal(err)
	}
	got := []any{read.Values["dataHome"], section(read.Values, "server")["port"], section(read.Values, "claude")["command"], section(read.Values, "claude")["model"], section(read.Values, "os")["editor"]}
	if !reflect.DeepEqual(got, []any{"/srv/lr", 4749.0, "/opt/claude", "claude-opus", "nvim"}) || !read.Exists {
		t.Errorf("the variables give %v", got)
	}
	for key, name := range map[string]string{"dataHome": "LAZYREVIEW_HOME", "server.port": "LAZYREVIEW_PORT", "claude.command": "LAZYREVIEW_CLAUDE", "claude.model": "LAZYREVIEW_MODEL", "os.editor": "LAZYREVIEW_EDITOR", "github.accounts": "default"} {
		if read.Sources[key] != name {
			t.Errorf("%s comes from %s, not %s", key, read.Sources[key], name)
		}
	}
	vars["LAZYREVIEW_PORT"] = ""
	if read, _ := ReadUserConfig(tr, env(vars), "darwin", "/home/lee"); section(read.Values, "server")["port"] != 4748.0 {
		t.Errorf("an empty variable set the port to %v", section(read.Values, "server")["port"])
	}
	vars["LAZYREVIEW_PORT"] = "x"
	if _, err := ReadUserConfig(tr, env(vars), "darwin", "/home/lee"); err == nil || err.Error() != `LAZYREVIEW_PORT must be a port number, not "x"` {
		t.Errorf("a port that is no number gave %v", err)
	}
}

func TestTheTerminalAppIsHandedTheEditorTheSettingThenVisualEditorAndVi(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	editor := func(setting string, vars map[string]string) string {
		read, _ := ConfigFrom(tr, "", "/home/lee/config.yml", "/home/lee")
		section(read.Values, "os")["editor"] = setting
		part, err := TerminalPart(tr, read, env(vars))
		if err != nil {
			t.Fatal(err)
		}
		return part.Editor
	}
	cases := []struct {
		setting string
		vars    map[string]string
		want    string
	}{
		{"", nil, "vi"}, {"", map[string]string{"EDITOR": "nano"}, "nano"}, {"", map[string]string{"EDITOR": "nano", "VISUAL": "code -w"}, "code -w"},
		{"hx", map[string]string{"EDITOR": "nano", "VISUAL": "code -w"}, "hx"},
	}
	for _, c := range cases {
		if got := editor(c.setting, c.vars); got != c.want {
			t.Errorf("%q with %v gives %q", c.setting, c.vars, got)
		}
	}
}
