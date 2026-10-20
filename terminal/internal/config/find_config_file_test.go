package config

import (
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

func only(files ...string) func(string) bool {
	return func(path string) bool {
		for _, f := range files {
			if f == path {
				return true
			}
		}
		return false
	}
}

func env(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

func found(t *testing.T, vars map[string]string, platform, home string, exists func(string) bool, want ConfigFile, why string) {
	t.Helper()
	got, err := FindConfigFile(i18n.EnglishTranslationSet(), env(vars), platform, home, exists)
	if err != nil || got != want {
		t.Errorf("%s: found %+v, %v, not %+v", why, got, err, want)
	}
}

func TestOnMacOSTheConfigIsInApplicationSupportUnlessXDGConfigHomeMovesIt(t *testing.T) {
	home := "/Users/lee"
	found(t, nil, "darwin", home, only(), ConfigFile{Path: "/Users/lee/Library/Application Support/lazyreview/config.yml"}, "the default")
	found(t, map[string]string{"XDG_CONFIG_HOME": "/Users/lee/.xdg"}, "darwin", home, only(), ConfigFile{Path: "/Users/lee/.xdg/lazyreview/config.yml"}, "a moved home")
	found(t, map[string]string{"XDG_CONFIG_HOME": "~/dots"}, "darwin", home, only(), ConfigFile{Path: "/Users/lee/dots/lazyreview/config.yml"}, "a folder from ~")
	found(t, map[string]string{"XDG_CONFIG_HOME": "relative"}, "darwin", home, only(), ConfigFile{Path: "/Users/lee/Library/Application Support/lazyreview/config.yml"}, "a folder that is not a full path is passed over")
}

func TestTheFirstConfigFoundWinsInTheXDGOrderAndTheDefaultIsWhereANewFileGoes(t *testing.T) {
	order := []string{
		"/Users/lee/Library/Application Support/lazyreview/config.yml",
		"/Users/lee/Library/Preferences/lazyreview/config.yml",
		"/Library/Application Support/lazyreview/config.yml",
		"/Library/Preferences/lazyreview/config.yml",
		"/Users/lee/.config/lazyreview/config.yml",
	}
	for i := range order {
		found(t, nil, "darwin", "/Users/lee", only(order[i:]...), ConfigFile{Path: order[i], Exists: true}, order[i])
	}
	found(t, map[string]string{"XDG_CONFIG_DIRS": "/etc/one:relative:/etc/two"}, "darwin", "/Users/lee", only("/etc/two/lazyreview/config.yml", order[4]),
		ConfigFile{Path: "/etc/two/lazyreview/config.yml", Exists: true}, "XDG_CONFIG_DIRS replaces the folders searched after the default")
}

func TestOnLinuxTheConfigIsInItsOwnConfigFolders(t *testing.T) {
	found(t, nil, "linux", "/home/lee", only(), ConfigFile{Path: "/home/lee/.config/lazyreview/config.yml"}, "the default")
	found(t, nil, "linux", "/home/lee", only("/etc/xdg/lazyreview/config.yml"), ConfigFile{Path: "/etc/xdg/lazyreview/config.yml", Exists: true}, "the system's")
}

func TestLazyreviewConfigDirNamesTheFolderAndConfigFileTheFileWhichMustExist(t *testing.T) {
	home := "/Users/lee"
	found(t, map[string]string{"LAZYREVIEW_CONFIG_DIR": "/tmp/lr"}, "darwin", home, only(), ConfigFile{Path: "/tmp/lr/config.yml"}, "a folder")
	found(t, map[string]string{"LAZYREVIEW_CONFIG_DIR": "/tmp/lr"}, "darwin", home, only("/tmp/lr/config.yml"), ConfigFile{Path: "/tmp/lr/config.yml", Exists: true}, "a folder's file")
	file := map[string]string{"LAZYREVIEW_CONFIG_FILE": "~/work.yml", "LAZYREVIEW_CONFIG_DIR": "/tmp/lr"}
	found(t, file, "darwin", home, only("/Users/lee/work.yml"), ConfigFile{Path: "/Users/lee/work.yml", Exists: true}, "a file")
	if _, err := FindConfigFile(i18n.EnglishTranslationSet(), env(file), "darwin", home, only()); err == nil || err.Error() != "LAZYREVIEW_CONFIG_FILE names /Users/lee/work.yml, which does not exist" {
		t.Errorf("a missing file gave %v", err)
	}
}
