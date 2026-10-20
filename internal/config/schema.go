package config

import (
	"regexp"
	"strings"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/utils"
)

// Patterns are what a colour, a login, a team's name, a hidden repository
// and a date must match.
var Patterns = struct {
	Colour, Login, Team, Repository, Date string
}{
	Colour:     `^#[0-9a-fA-F]{6}$`,
	Login:      `^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?$`,
	Team:       `^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?/[A-Za-z0-9_.-]+$`,
	Repository: `^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?/(?:[A-Za-z0-9_.-]+|\*)$`,
	Date:       `^\d{4}-\d{2}-\d{2}$`,
}

var (
	colourPattern     = regexp.MustCompile(Patterns.Colour)
	loginPattern      = regexp.MustCompile(Patterns.Login)
	teamPattern       = regexp.MustCompile(Patterns.Team)
	repositoryPattern = regexp.MustCompile(Patterns.Repository)
	datePattern       = regexp.MustCompile(Patterns.Date)
)

var numbered = []string{"jumpToRepository", "jumpToBlock", "choose"}

// Schema is every setting of lazyreview's config, once: its default, its
// check, the variable that overrides it, and the words that say what it
// does, as lazygit keeps its own in GetDefaultConfig. The reader, the
// documentation and the editors' schema are all made from it.
func Schema(tr *i18n.TranslationSet) *Section {
	lines := func(text string) []string { return strings.Split(text, "\n") }
	setting := func(check string, value any, env, about string) Entry {
		s := &Setting{Check: check, Default: value, Env: env}
		if about != "" {
			s.About = lines(about)
		}
		return Entry{Setting: s}
	}
	named := func(name string, e Entry) Entry {
		e.Name = name
		return e
	}
	accounts := setting("accounts", []Account{}, "", tr.ConfigAboutAccounts)
	accounts.Setting.Example = lines(tr.ConfigAccountsExample)
	hidden := setting("repositories", []string{}, "", tr.ConfigAboutHiddenRepositories)
	hidden.Setting.Example = lines(tr.ConfigHiddenRepositoriesExample)
	lineNumbers := setting("choice", "absolute", "", tr.ConfigAboutLineNumbers)
	lineNumbers.Setting.Choices = []string{"absolute", "relative", "none"}
	return &Section{Entries: []Entry{
		named("dataHome", setting("folder", "~/.lazyreview", "LAZYREVIEW_HOME", tr.ConfigAboutDataHome)),
		{Name: "claude", Section: &Section{Entries: []Entry{
			named("command", setting("command", "claude", "LAZYREVIEW_CLAUDE", tr.ConfigAboutClaudeCommand)),
			named("model", setting("text", "", "LAZYREVIEW_MODEL", tr.ConfigAboutClaudeModel)),
		}}},
		{Name: "os", Section: &Section{Entries: []Entry{named("editor", setting("text", "", "LAZYREVIEW_EDITOR", tr.ConfigAboutEditor))}}},
		{Name: "github", Section: &Section{Entries: []Entry{named("accounts", accounts), named("hiddenRepositories", hidden)}}},
		{Name: "gui", Section: &Section{Entries: []Entry{
			named("lineNumbers", lineNumbers),
			{Name: "theme", Section: &Section{About: lines(tr.ConfigAboutTheme), Entries: []Entry{
				{Name: "dark", Section: colours(darkColours)},
				{Name: "light", Section: colours(lightColours)},
			}}},
		}}},
		{Name: "keybinding", Section: keybindings(tr)},
	}}
}

// Retired is each setting lazyreview no longer has, by its dotted key, with
// what does its work now.
func Retired(tr *i18n.TranslationSet) map[string]string {
	move := func(side string) string {
		return utils.ResolvePlaceholderString(tr.ConfigRetiredDiagramPanelMove, map[string]string{"side": side})
	}
	return map[string]string{"keybinding.diagramPanel.left": move("left"), "keybinding.diagramPanel.right": move("right")}
}

// Unused is each setting a config may still hold that does nothing now, by
// its dotted key, with why, so a config that set it still reads.
func Unused(tr *i18n.TranslationSet) map[string]string {
	return map[string]string{"keybinding.universal.web": tr.ConfigUnusedWeb, "server": tr.ConfigUnusedServer}
}

// Defaults is the default config of a section, as values: each section a
// map by name, and each setting its default.
func Defaults(s *Section) map[string]any {
	out := map[string]any{}
	for _, e := range s.Entries {
		if e.Section != nil {
			out[e.Name] = Defaults(e.Section)
		} else {
			out[e.Name] = copied(e.Setting.Default)
		}
	}
	return out
}

// Leaves is every setting of a section with its dotted key, in order.
func Leaves(s *Section, prefix string) []Leaf {
	var out []Leaf
	for _, e := range s.Entries {
		key := e.Name
		if prefix != "" {
			key = prefix + "." + e.Name
		}
		if e.Section != nil {
			out = append(out, Leaves(e.Section, key)...)
		} else {
			out = append(out, Leaf{Key: key, Setting: e.Setting})
		}
	}
	return out
}

func copied(v any) any {
	switch x := v.(type) {
	case []string:
		return append([]string{}, x...)
	case []Account:
		return append([]Account{}, x...)
	}
	return v
}

func colours(given namedColours) *Section {
	section := &Section{}
	for _, c := range given.roles {
		section.Entries = append(section.Entries, Entry{Name: c.name, Setting: &Setting{Check: "colour", Default: c.value}})
	}
	section.Entries = append(section.Entries, Entry{Name: "pullRequestColors", Setting: &Setting{Check: "colours", Default: given.pulls}})
	components := &Section{Flow: true}
	for _, c := range given.components {
		components.Entries = append(components.Entries, Entry{Name: c.name, Setting: &Setting{Check: "colour", Default: c.value}})
	}
	return &Section{Entries: append(section.Entries, Entry{Name: "componentColors", Section: components})}
}

func keys(note string, actions ...keyDefault) *Section {
	section := &Section{Note: note}
	for _, a := range actions {
		check := "keys"
		for _, n := range numbered {
			if n == a.action {
				check = "numberedKeys"
			}
		}
		section.Entries = append(section.Entries, Entry{Name: a.action, Setting: &Setting{Check: check, Default: a.keys}})
	}
	return section
}

// IsRepository says whether name is a hidden repository's entry: a
// repository as owner/name, or owner/* for every repository of the owner.
func IsRepository(name string) bool {
	return repositoryPattern.MatchString(name)
}
