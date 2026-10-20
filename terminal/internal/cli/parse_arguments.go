package cli

import (
	"slices"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const anyCount = -1

var configCommands = []string{"path", "defaults", "check", "move"}

type command struct {
	valued []string
	flags  []string
	least  int
	most   int
	check  func(tr *i18n.TranslationSet, o Options) string
}

var commands = map[string]command{
	"load":     {valued: []string{"--account", "--config", "--repo", "--base", "--head", "--title"}, flags: []string{"--context", "--no-context"}, most: anyCount, check: checkLoad},
	"attach":   {least: 2, most: 2},
	"list":     {most: 0},
	"terminal": {most: 1},
	"config":   {flags: []string{"--check"}, most: 1, check: checkConfig},
	"start":    {most: 0},
	"stop":     {most: 0},
	"help":     {most: 0},
}

// ParseArguments reads the command line, argv without the program's name:
// a command, then its options and arguments in any order. No command, or
// --help or -h anywhere, is help. It fails saying what was wrong, followed
// by the usage.
func ParseArguments(tr *i18n.TranslationSet, argv []string) (Options, error) {
	fail := func(message string) (Options, error) {
		return Options{}, models.Refused(message + "\n\n" + tr.Usage)
	}
	name := "help"
	if len(argv) > 0 {
		name = argv[0]
	}
	if name == "--help" || name == "-h" {
		name = "help"
	}
	spec, known := commands[name]
	if !known {
		return fail(utils.ResolvePlaceholderString(tr.CliUnknownCommand, map[string]string{"command": name}))
	}
	o := Options{Command: name}
	rest := argv[min(1, len(argv)):]
	for i := 0; i < len(rest); i++ {
		a := rest[i]
		switch {
		case a == "--help" || a == "-h":
			return Options{Command: "help"}, nil
		case slices.Contains(spec.flags, a):
			o.setFlag(a)
		case slices.Contains(spec.valued, a):
			if i+1 >= len(rest) || strings.HasPrefix(rest[i+1], "--") {
				return fail(utils.ResolvePlaceholderString(tr.CliNeedsValue, map[string]string{"option": a}))
			}
			i++
			o.setValue(a, rest[i])
		case strings.HasPrefix(a, "--"):
			return fail(utils.ResolvePlaceholderString(tr.CliUnknownOption, map[string]string{"option": a, "command": name}))
		default:
			o.Positional = append(o.Positional, a)
		}
	}
	if spec.least > 0 && len(o.Positional) < spec.least {
		return fail(utils.ResolvePlaceholderString(tr.CliNeedsArguments, map[string]string{"command": name, "count": strconv.Itoa(spec.least)}))
	}
	if spec.most != anyCount && len(o.Positional) > spec.most {
		arguments := tr.CliNoArguments
		if spec.most > 0 {
			arguments = utils.Plural(spec.most, tr.ArgumentNoun)
		}
		return fail(utils.ResolvePlaceholderString(tr.CliTakesArguments, map[string]string{"command": name, "arguments": arguments}))
	}
	if spec.check != nil {
		if message := spec.check(tr, o); message != "" {
			return fail(message)
		}
	}
	return o, nil
}

func (o *Options) setFlag(flag string) {
	switch flag {
	case "--context":
		o.Context = true
	case "--no-context":
		o.NoContext = true
	case "--check":
		o.Check = true
	}
}

func (o *Options) setValue(option, value string) {
	given := &value
	switch option {
	case "--account":
		o.Account = given
	case "--config":
		o.Config = given
	case "--repo":
		o.Repo = given
	case "--base":
		o.Base = given
	case "--head":
		o.Head = given
	case "--title":
		o.Title = given
	}
}

func checkLoad(tr *i18n.TranslationSet, o Options) string {
	if o.Repo == nil {
		for _, stray := range []struct {
			option string
			value  *string
		}{{"--base", o.Base}, {"--head", o.Head}, {"--title", o.Title}} {
			if stray.value != nil {
				return utils.ResolvePlaceholderString(tr.CliNeedsRepo, map[string]string{"option": stray.option})
			}
		}
		if len(o.Positional) == 0 {
			return tr.CliLoadNeedsSomething
		}
		return ""
	}
	if len(o.Positional) > 0 {
		return tr.CliNotBoth
	}
	if o.Account != nil {
		return tr.CliAccountForPulls
	}
	if o.Base == nil {
		return utils.ResolvePlaceholderString(tr.CliRepoNeeds, map[string]string{"option": "--base"})
	}
	if o.Head == nil {
		return utils.ResolvePlaceholderString(tr.CliRepoNeeds, map[string]string{"option": "--head"})
	}
	return ""
}

func checkConfig(tr *i18n.TranslationSet, o Options) string {
	what := ""
	if len(o.Positional) > 0 {
		what = o.Positional[0]
	}
	if !slices.Contains(configCommands, what) {
		if what == "" {
			return tr.CliConfigNeeds
		}
		return utils.ResolvePlaceholderString(tr.CliConfigNeedsNot, map[string]string{"given": what})
	}
	if o.Check && what != "move" {
		return tr.CliCheckIsForMove
	}
	return ""
}
