// Package app starts the terminal app: it builds what every layer shares,
// the reviews it reads and changes, and the gui, then runs Bubble Tea until
// the app quits.
package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/connorb645/lazyreview/terminal/internal/commands/github"
	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/common"
	"github.com/connorb645/lazyreview/terminal/internal/config"
	"github.com/connorb645/lazyreview/terminal/internal/gui"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/queue"
	"github.com/connorb645/lazyreview/terminal/internal/reviews"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
)

// Run opens the app on the review id names, or with none on the home
// screen, as a client of the server whose record is at serverFile, with
// the part of the config lazyreview hands over as JSON, and returns once
// the app quits.
func Run(serverFile, userConfig, review string) error {
	tr := i18n.EnglishTranslationSet()
	given, err := config.Decode(userConfig, tr)
	if err != nil {
		return fmt.Errorf("%s: %w", tr.UserConfigUnreadable, err)
	}
	if !filepath.IsAbs(given.DataHome) {
		return errors.New(tr.NoDataHome)
	}
	cmn := &common.Common{Tr: tr, Config: given, Theme: terminalTheme(given.Theme)}
	reviewStore := store.NewReviewStore(tr, store.DataHome{Root: given.DataHome}, store.NewLocks(tr))
	gh := func(host, account string) *github.GitHub { return github.New(tr, host, account, nil) }
	now := func() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") }
	all := reviews.New(tr, reviewStore, queue.NewService(tr, reviewStore, gh, now), server.NewClient(cmn, serverFile))
	opts := gui.Options{Reviews: all, Review: review, Editor: given.Editor}
	program := tea.NewProgram(gui.New(cmn, opts), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = program.Run()
	return err
}

func terminalTheme(colours config.ThemeConfig) theme.Theme {
	if lipgloss.HasDarkBackground() {
		return theme.New(colours.Dark, true, lipgloss.ColorProfile())
	}
	return theme.New(colours.Light, false, lipgloss.ColorProfile())
}
