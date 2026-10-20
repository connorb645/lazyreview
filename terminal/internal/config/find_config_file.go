package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const (
	configName   = "config.yml"
	configFolder = "lazyreview"
)

// FindConfigFile is where lazyreview's config file is, as lazygit finds its
// own: LAZYREVIEW_CONFIG_FILE, the file itself, which must exist; else
// config.yml in LAZYREVIEW_CONFIG_DIR, used as it is; else the first
// lazyreview/config.yml in the platform's config folders, in the order the
// XDG base folders give them; else the platform's default, where a new file
// goes. getenv reads a variable, home is the user's home folder and exists
// says whether a path is a file.
func FindConfigFile(tr *i18n.TranslationSet, getenv func(string) string, platform, home string, exists func(string) bool) (ConfigFile, error) {
	if given := getenv("LAZYREVIEW_CONFIG_FILE"); given != "" {
		file := absolutePath(expandHome(given, home))
		if !exists(file) {
			return ConfigFile{}, errors.New(utils.ResolvePlaceholderString(tr.ConfigFileMissing, map[string]string{"path": file}))
		}
		return ConfigFile{Path: file, Exists: true}, nil
	}
	if given := getenv("LAZYREVIEW_CONFIG_DIR"); given != "" {
		file := filepath.Join(absolutePath(expandHome(given, home)), configName)
		return ConfigFile{Path: file, Exists: exists(file)}, nil
	}
	folders := configFolders(getenv, platform, home)
	for _, folder := range folders {
		if file := filepath.Join(folder, configFolder, configName); exists(file) {
			return ConfigFile{Path: file, Exists: true}, nil
		}
	}
	return ConfigFile{Path: filepath.Join(folders[0], configFolder, configName)}, nil
}

// IsFile says whether path is a file.
func IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func configFolders(getenv func(string) string, platform, home string) []string {
	one := func(name, fallback string) string {
		if given := absolutes([]string{getenv(name)}, home); len(given) > 0 {
			return given[0]
		}
		return fallback
	}
	list := func(name string, fallbacks []string) []string {
		if given := absolutes(strings.Split(getenv(name), string(filepath.ListSeparator)), home); len(given) > 0 {
			return given
		}
		return absolutes(fallbacks, home)
	}
	switch platform {
	case "windows":
		local := or(getenv("LOCALAPPDATA"), filepath.Join(home, "AppData", "Local"))
		roaming := or(getenv("APPDATA"), filepath.Join(home, "AppData", "Roaming"))
		shared := or(getenv("PROGRAMDATA"), `C:\ProgramData`)
		return unique(append([]string{one("XDG_CONFIG_HOME", local)}, list("XDG_CONFIG_DIRS", []string{shared, roaming})...))
	case "darwin":
		support := filepath.Join(home, "Library", "Application Support")
		return unique(append([]string{one("XDG_CONFIG_HOME", support)}, list("XDG_CONFIG_DIRS", []string{
			filepath.Join(home, "Library", "Preferences"), "/Library/Application Support", "/Library/Preferences", filepath.Join(home, ".config"),
		})...))
	}
	return unique(append([]string{one("XDG_CONFIG_HOME", filepath.Join(home, ".config"))}, list("XDG_CONFIG_DIRS", []string{"/etc/xdg"})...))
}

func absolutes(paths []string, home string) []string {
	var out []string
	for _, p := range paths {
		if p == "" {
			continue
		}
		if p = expandHome(p, home); filepath.IsAbs(p) {
			out = append(out, p)
		}
	}
	return out
}

func expandHome(given, home string) string {
	switch {
	case given == "~":
		return home
	case strings.HasPrefix(given, "~/"), strings.HasPrefix(given, `~\`):
		return filepath.Join(home, given[2:])
	}
	return given
}

func absolutePath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

func or(given, fallback string) string {
	if given != "" {
		return given
	}
	return fallback
}

func unique(paths []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}
