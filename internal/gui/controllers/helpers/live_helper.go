package helpers

import (
	stdcontext "context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

const liveTries = 5

// LiveHelper hears the changes made to the review open, and loads what
// changed.
type LiveHelper struct {
	c *HelperCommon
}

// Listen hears a review's changes as messages until ctx ends, trying again a few times, each after a longer wait, when the
// stream breaks, then telling that it stopped.
func Listen(ctx stdcontext.Context, c types.Reviews, id string) <-chan tea.Msg {
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
		notices := make(chan models.Notice)
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

// ListenTo is the state hearing a review's changes, and the command that
// waits for the first, unless it hears them already.
func (self *LiveHelper) ListenTo(s context.ContextTree, id string) (context.ContextTree, tea.Cmd) {
	if s.LiveFor == id {
		return s, nil
	}
	if s.StopLive != nil {
		s.StopLive()
	}
	ctx, stop := stdcontext.WithCancel(stdcontext.Background())
	s.Live, s.StopLive, s.LiveFor = self.c.Listen(ctx, self.c.Reviews, id), stop, id
	return s, nextNotice(s.Live)
}

// AfterNotice is the state once a change to the review is heard: the review
// or its state loading again, or a notice that a draft holds it back.
func (self *LiveHelper) AfterNotice(s context.ContextTree, msg types.LiveNotice) (context.ContextTree, tea.Cmd) {
	next := nextNotice(s.Live)
	if s.Review == nil || msg.ID != s.Review.ID {
		return s, next
	}
	var cmds []tea.Cmd
	switch n := msg.Notice; {
	case n.Build != "" && n.Build != s.Review.Page.Build && (s.EditingNow || len(s.Drafts) > 0):
		s = s.Said(utils.ResolvePlaceholderString(self.c.Tr.LoadedElsewhere, map[string]string{"refresh": self.c.KeyLabels(s.Place()).Of("refresh")}), false)
	case n.Build != "" && n.Build != s.Review.Page.Build:
		cmds = append(cmds, self.c.Helpers.Review.Reload(s.Review.ID, false))
	case n.State != "" && n.State != s.Review.State.Version:
		cmds = append(cmds, self.c.Helpers.Review.LoadState(s.Review.ID))
	}
	return s, tea.Batch(append(cmds, next)...)
}

// AfterStopped is the state once the review's changes can no longer be
// heard, saying why.
func (self *LiveHelper) AfterStopped(s context.ContextTree, msg types.LiveStopped) context.ContextTree {
	if s.Review == nil || msg.ID != s.Review.ID {
		return s
	}
	s.LiveFor = ""
	return s.Said(utils.ResolvePlaceholderString(self.c.Tr.StoppedHearing, map[string]string{"error": msg.Err.Error(), "refresh": self.c.KeyLabels(s.Place()).Of("refresh")}), true)
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
