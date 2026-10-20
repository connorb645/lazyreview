// Command lazyreview-terminal is lazyreview in a terminal: its reviews, with
// Vim keys. lazyreview terminal builds it and runs it. It also loads a
// review, answers a question with Claude, as a process of its own, and is
// the guard Claude Code asks about each tool call while it answers.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/connorb645/lazyreview/terminal/internal/app"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/loader"
)

func main() {
	tr := i18n.EnglishTranslationSet()
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "load":
			load(tr, os.Args[2:])
			return
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
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, tr.Usage)
	}
	flag.Parse()
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := app.Run(flag.Arg(0)); err != nil {
		fmt.Fprintln(os.Stderr, tr.TerminalFailed+":", err)
		os.Exit(1)
	}
}

func load(tr *i18n.TranslationSet, args []string) {
	flags := flag.NewFlagSet("load", flag.ExitOnError)
	account := flags.String("account", "", tr.AccountFlag)
	configFile := flags.String("config", "", tr.ConfigFlag)
	repo := flags.String("repo", "", tr.RepoFlag)
	base := flags.String("base", "", tr.BaseFlag)
	head := flags.String("head", "", tr.HeadFlag)
	title := flags.String("title", "", tr.TitleFlag)
	withContext := flags.Bool("context", false, tr.ContextFlag)
	flags.Bool("no-context", false, tr.NoContextFlag)
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, tr.Usage)
	}
	_ = flags.Parse(args)
	given := map[string]bool{}
	flags.Visit(func(f *flag.Flag) { given[f.Name] = true })
	pulls := loader.PullsRequest{Links: flags.Args()}
	if given["account"] {
		pulls.Account = account
	}
	if given["context"] || given["no-context"] {
		drawn := *withContext
		pulls.Context = &drawn
	}
	if given["config"] {
		text, err := os.ReadFile(*configFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		pulls.Config = text
	}
	var branch *loader.BranchRequest
	if given["repo"] {
		branch = &loader.BranchRequest{Repo: *repo, Base: *base, Head: *head, Config: pulls.Config, Context: pulls.Context}
		if given["title"] {
			branch.Title = title
		}
	}
	if err := app.Load(os.Stdout, pulls, branch); err != nil {
		fmt.Fprintln(os.Stderr, err)
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
