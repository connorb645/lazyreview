// Package app is lazyreview's command: it does what the command line asks,
// building what each command uses, and for the terminal app what every
// layer shares, the reviews it reads and changes, and the gui, then runs
// Bubble Tea until the app quits.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/connorb645/lazyreview/internal/ask"
	"github.com/connorb645/lazyreview/internal/commands/claude"
	"github.com/connorb645/lazyreview/internal/commands/git"
	"github.com/connorb645/lazyreview/internal/commands/github"
	"github.com/connorb645/lazyreview/internal/commands/oscommands"
	"github.com/connorb645/lazyreview/internal/common"
	"github.com/connorb645/lazyreview/internal/config"
	"github.com/connorb645/lazyreview/internal/gui"
	"github.com/connorb645/lazyreview/internal/home"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/loader"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/queue"
	"github.com/connorb645/lazyreview/internal/reviews"
	"github.com/connorb645/lazyreview/internal/store"
	"github.com/connorb645/lazyreview/internal/theme"
	"github.com/connorb645/lazyreview/internal/utils"
)

// Run opens the app on the review id names, or with none on the home
// screen, with the settings given, reading the config again with
// readConfig where the home screen needs it, and returns once the app
// quits.
func Run(tr *i18n.TranslationSet, given config.UserConfig, readConfig func() (config.Read, error), review string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmn := &common.Common{Tr: tr, Config: given, Theme: terminalTheme(given.Theme)}
	locks := store.NewLocks(tr)
	reviewStore := store.NewReviewStore(tr, store.DataHome{Root: given.DataHome}, locks)
	answerer := func(ref models.ReviewRef, conversation int) error {
		return oscommands.RunDetached(self, []string{"answer", "--data-home", given.DataHome, "--claude", given.Claude.Command, "--model", given.Claude.Model, ref.ID(), strconv.Itoa(conversation)})
	}
	asking := ask.NewService(tr, reviewStore, locks, claude.New(tr, given.Claude.Command, given.Claude.Model), nil, nil, answerer, nil, isoNow)
	all := reviews.New(tr, reviewStore, queue.NewService(tr, reviewStore, gitHubFor(tr), isoNow), asking,
		home.NewOpenPulls(tr, reviewStore, readConfig, gitHubFor(tr)), home.NewHiddenRepositories(tr, readConfig),
		loader.NewLoader(tr, reviewStore, locks, git.NewClones(tr, nil), gitHubFor(tr), isoNow), loader.NewBranchLoader(tr, reviewStore, isoNow))
	opts := gui.Options{Reviews: all, Review: review, Editor: given.Editor}
	program := tea.NewProgram(gui.New(cmn, opts), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = program.Run()
	return err
}

// Load loads pull requests into a review, or with branch given a branch,
// in the data home dataHome, and returns the review it loaded, what it
// drew and the notes on what it found. A signal to end stops a load of
// pull requests, all or nothing.
func Load(tr *i18n.TranslationSet, dataHome string, pulls loader.PullsRequest, branch *loader.BranchRequest) (loader.Loaded, error) {
	locks := store.NewLocks(tr)
	reviewStore := store.NewReviewStore(tr, store.DataHome{Root: dataHome}, locks)
	if branch != nil {
		return loader.NewBranchLoader(tr, reviewStore, isoNow).Load(*branch)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer stop()
	return loader.NewLoader(tr, reviewStore, locks, git.NewClones(tr, nil), gitHubFor(tr), isoNow).Load(ctx, pulls)
}

// Answer has Claude answer conversation's last question in the review id
// names, in the data home dataHome, asking command, Claude Code, with
// model, and returns once the answer, or why there is none, is kept. The
// terminal app runs it as a process of its own, so an answer goes on when
// the app quits. A signal to end stops Claude, and says so.
func Answer(dataHome, command, model, review string, conversation int) error {
	tr := i18n.EnglishTranslationSet()
	if !filepath.IsAbs(dataHome) {
		return errors.New(tr.NoDataHome)
	}
	ref, err := models.ParseReviewRef(tr, review)
	if err != nil {
		return err
	}
	locks := store.NewLocks(tr)
	reviewStore := store.NewReviewStore(tr, store.DataHome{Root: dataHome}, locks)
	asking := ask.NewService(tr, reviewStore, locks, claude.New(tr, command, model), nil, nil, nil, nil, isoNow)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer stop()
	return asking.Answer(ctx, ref, conversation)
}

// Guard answers Claude Code's PreToolUse hook, while Claude answers in the
// copy of the code in folder copy: it reads the call from input, writes
// the answer that allows or refuses it to stdout, or why it could not read
// it to stderr, and returns the exit code Claude Code reads.
func Guard(copy string, input io.Reader, stdout, stderr io.Writer) int {
	tr := i18n.EnglishTranslationSet()
	call, err := io.ReadAll(input)
	if err != nil {
		fmt.Fprint(stderr, utils.ResolvePlaceholderString(tr.GuardFailed, map[string]string{"why": err.Error()}))
		return 2
	}
	output, code := ask.HookAnswer(tr, call, copy)
	if code == 0 {
		fmt.Fprint(stdout, output)
	} else {
		fmt.Fprint(stderr, output)
	}
	return code
}

func isoNow() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

func terminalTheme(colours config.ThemeConfig) theme.Theme {
	if lipgloss.HasDarkBackground() {
		return theme.New(colours.Dark, true, lipgloss.ColorProfile())
	}
	return theme.New(colours.Light, false, lipgloss.ColorProfile())
}

func settings(tr *i18n.TranslationSet, readConfig func() (config.Read, error), getenv func(string) string) (config.UserConfig, error) {
	read, err := readConfig()
	if err != nil {
		return config.UserConfig{}, err
	}
	given, err := config.TerminalPart(tr, read, getenv)
	if err != nil {
		return config.UserConfig{}, err
	}
	if !filepath.IsAbs(given.DataHome) {
		return config.UserConfig{}, errors.New(tr.NoDataHome)
	}
	return given, nil
}

func gitHubFor(tr *i18n.TranslationSet) func(host, account string) *github.GitHub {
	return func(host, account string) *github.GitHub { return github.New(tr, host, account, nil) }
}
