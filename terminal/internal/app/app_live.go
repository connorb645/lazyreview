package app

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"review-desk/terminal/internal/server"
)

const liveTries = 5

func listen(ctx context.Context, c *server.Client, id string) <-chan tea.Msg {
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
					if !give(liveNotice{id: id, notice: n}) {
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
				give(liveStopped{id: id, err: err})
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
	if a.liveFor == id {
		return a, nil
	}
	if a.stopLive != nil {
		a.stopLive()
	}
	ctx, stop := context.WithCancel(context.Background())
	a.live, a.stopLive, a.liveFor = a.opts.Listen(ctx, a.opts.Server, id), stop, id
	return a, nextNotice(a.live)
}

func (a App) afterNotice(msg liveNotice) (tea.Model, tea.Cmd) {
	next := nextNotice(a.live)
	if a.review == nil || msg.id != a.review.id {
		return a, next
	}
	var cmds []tea.Cmd
	switch n := msg.notice; {
	case n.Build != "" && n.Build != a.review.page.Build && (a.editingNow || len(a.drafts) > 0):
		a = a.say("This review was loaded again. R loads it here; your drafts stay.", false)
	case n.Build != "" && n.Build != a.review.page.Build:
		cmds = append(cmds, reloadReview(a.opts.Server, a.review.id, false))
	case n.State != "" && n.State != a.review.state.Version:
		cmds = append(cmds, loadState(a.opts.Server, a.review.id))
	}
	return a, tea.Batch(append(cmds, next)...)
}

func reloadReview(c *server.Client, id string, refresh bool) tea.Cmd {
	return func() tea.Msg {
		var notes []string
		if refresh {
			var err error
			if notes, err = c.Refresh(id); err != nil {
				return reviewReloaded{err: err}
			}
		}
		loaded, ok := loadReview(c, id)().(reviewLoaded)
		if !ok || loaded.err != nil {
			return reviewReloaded{err: loaded.err}
		}
		return reviewReloaded{data: loaded.data, notes: notes}
	}
}

func (a App) afterReload(msg reviewReloaded) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return a.say(msg.err.Error(), true), nil
	}
	if a.review == nil || msg.data.id != a.review.id {
		return a, nil
	}
	words := "Loaded again."
	if len(msg.notes) > 0 {
		words += " " + strings.Join(msg.notes, " ")
	}
	a.review = &msg.data
	a.files = map[string]server.File{}
	switch a.screen {
	case screenDiff:
		next, cmd := a.openPlace(a.diff.component, a.diff.place())
		return next.(App).say(words, false), cmd
	case screenDiagram:
		return a.showBox(a.dg.place()).say(words, false), nil
	}
	return a.relaid().say(words, false), nil
}
