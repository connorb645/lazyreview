// Command lazyreview-terminal is lazyreview in a terminal: its reviews, with
// Vim keys. lazyreview terminal builds it and runs it. It also answers a
// question with Claude, as a process of its own, and is the guard Claude
// Code asks about each tool call while it answers.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/connorb645/lazyreview/terminal/internal/app"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
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
	serverFile := flag.String("server-file", "", tr.ServerFileFlag)
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, tr.Usage)
	}
	flag.Parse()
	if *serverFile == "" || flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := app.Run(*serverFile, flag.Arg(0)); err != nil {
		fmt.Fprintln(os.Stderr, tr.TerminalFailed+":", err)
		os.Exit(1)
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
		fmt.Fprintln(os.Stderr, tr.TerminalFailed+":", err)
		os.Exit(1)
	}
}
