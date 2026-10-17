package gui

import (
	stdcontext "context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

const liveTries = 5

func listen(ctx stdcontext.Context, c *server.Client, id string) <-chan tea.Msg {
	out := make(chan tea.Msg)
	give := func(msg tea.Msg) bool {
		select {
		case out <- msg:
			return true
		case <-ctx.Done():
			return false
		}
	}
	go func() {
		defer close(out)
		notices := make(chan server.Notice)
		failures := 0
		for {
			stopped := make(chan error, 1)
			go func() { stopped <- c.Events(ctx, id, notices) }()
			var err error
		stream:
			for {
				select {
				case n := <-notices:
					failures = 0
					if !give(types.LiveNotice{ID: id, Notice: n}) {
						return
					}
				case err = <-stopped:
					break stream
				}
			}
			if ctx.Err() != nil {
				return
			}
			failures++
			if failures >= liveTries {
				give(types.LiveStopped{ID: id, Err: err})
				return
			}
			select {
			case <-time.After(time.Duration(1<<(failures-1)) * time.Second):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func nextNotice(live <-chan tea.Msg) tea.Cmd {
	if live == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-live
		if !ok {
			return nil
		}
		return msg
	}
}

func (a App) listenTo(id string) (App, tea.Cmd) {
	if a.LiveFor == id {
		return a, nil
	}
	if a.StopLive != nil {
		a.StopLive()
	}
	ctx, stop := stdcontext.WithCancel(stdcontext.Background())
	a.Live, a.StopLive, a.LiveFor = a.opts.Listen(ctx, a.opts.Server, id), stop, id
	return a, nextNotice(a.Live)
}

func (a App) afterNotice(msg types.LiveNotice) (tea.Model, tea.Cmd) {
	next := nextNotice(a.Live)
	if a.Review == nil || msg.ID != a.Review.ID {
		return a, next
	}
	var cmds []tea.Cmd
	switch n := msg.Notice; {
	case n.Build != "" && n.Build != a.Review.Page.Build && (a.EditingNow || len(a.Drafts) > 0):
		a = a.say(a.c.Tr.LoadedElsewhere, false)
	case n.Build != "" && n.Build != a.Review.Page.Build:
		cmds = append(cmds, reloadReview(a.c.Tr, a.opts.Server, a.Review.ID, false))
	case n.State != "" && n.State != a.Review.State.Version:
		cmds = append(cmds, loadState(a.opts.Server, a.Review.ID))
	}
	return a, tea.Batch(append(cmds, next)...)
}

func reloadReview(tr *i18n.TranslationSet, c *server.Client, id string, refresh bool) tea.Cmd {
	return func() tea.Msg {
		var notes []string
		if refresh {
			var err error
			if notes, err = c.Refresh(id); err != nil {
				return types.ReviewReloaded{Err: err}
			}
		}
		loaded, ok := loadReview(tr, c, id)().(types.ReviewLoaded)
		if !ok || loaded.Err != nil {
			return types.ReviewReloaded{Err: loaded.Err}
		}
		return types.ReviewReloaded{Data: loaded.Data, Notes: notes}
	}
}

func (a App) afterReload(msg types.ReviewReloaded) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		return a.say(msg.Err.Error(), true), nil
	}
	if a.Review == nil || msg.Data.ID != a.Review.ID {
		return a, nil
	}
	words := a.c.Tr.LoadedAgain
	if len(msg.Notes) > 0 {
		words += " " + strings.Join(msg.Notes, " ")
	}
	a.Review = &msg.Data
	a.Files = map[string]server.File{}
	a = a.keptFilter()
	switch a.Screen() {
	case context.ScreenDiff:
		next, cmd := a.openPlace(a.Diff.Component, a.Diff.Place())
		return next.(App).say(words, false), cmd
	case context.ScreenDiagram:
		return a.showBox(a.Diagram.Place()).say(words, false), nil
	}
	return a.relaid().say(words, false), nil
}
