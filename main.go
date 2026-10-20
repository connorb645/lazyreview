// Command lazyreview reviews GitHub pull requests on this machine, in a
// terminal. bin/lazyreview builds it when its code changed, then runs it,
// and `lazyreview help` lists its commands. It also answers a question with
// Claude, as a process of its own, and is the guard Claude Code asks about
// each tool call while it answers.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/connorb645/lazyreview/internal/app"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/utils"
)

func main() {
	tr := i18n.EnglishTranslationSet()
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "answer":
			answer(tr, os.Args[2:])
			return
		case "guard":
			if len(os.Args) != 3 {
				fmt.Fprintln(os.Stderr, tr.Usage)
				os.Exit(2)
			}
			os.Exit(app.Guard(os.Args[2], os.Stdin, os.Stdout, os.Stderr))
		}
	}
	if err := app.NewCommandLine(tr, os.Stdout).Run(os.Args[1:]); err != nil {
		failed(tr, err)
	}
}

func answer(tr *i18n.TranslationSet, args []string) {
	flags := flag.NewFlagSet("answer", flag.ExitOnError)
	dataHome := flags.String("data-home", "", tr.DataHomeFlag)
	command := flags.String("claude", "", tr.ClaudeFlag)
	model := flags.String("model", "", tr.ModelFlag)
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, tr.Usage)
	}
	_ = flags.Parse(args)
	conversation, err := strconv.Atoi(flags.Arg(1))
	if *dataHome == "" || flags.NArg() != 2 || err != nil {
		flags.Usage()
		os.Exit(2)
	}
	if err := app.Answer(*dataHome, *command, *model, flags.Arg(0), conversation); err != nil {
		failed(tr, err)
	}
}

func failed(tr *i18n.TranslationSet, err error) {
	fmt.Fprintln(os.Stderr, utils.ResolvePlaceholderString(tr.CliFailed, map[string]string{"why": err.Error()}))
	os.Exit(1)
}
