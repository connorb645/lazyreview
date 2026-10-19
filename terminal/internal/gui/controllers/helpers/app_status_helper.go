package helpers

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var spinner = []string{"|", "/", "-", "\\"}

const spinRate = 100 * time.Millisecond

// AppStatusHelper is the bar's word on work under way that no key clears:
// Claude answering, with a spinner after it, as lazygit shows its own work
// (lazygit:`pkg/gui/status/status_manager.go:78`), in lazygit's frames
// (lazygit:`pkg/config/user_config.go:924-927`). It turns at half lazygit's
// rate, as each frame draws the whole screen again.
type AppStatusHelper struct {
	c *HelperCommon
}

// Status is what the bar says while Claude answers a question asked here,
// or one in the review that waits for an answer, asked in either app, with
// the latest thing it did, cut to fit `room` columns so the spinner always
// shows; or nothing.
func (self *AppStatusHelper) Status(s context.ContextTree, room int) string {
	if !answering(s) {
		return ""
	}
	words := self.c.Tr.ClaudeIsWorking
	if step := latestStep(s); step != "" {
		words = utils.ResolvePlaceholderString(self.c.Tr.ClaudeIsWorkingOn, map[string]string{"step": step})
	}
	frame := spinner[s.Spin%len(spinner)]
	return ansi.Truncate(words, max(1, room-len(frame)-1), "…") + " " + frame
}

// Kept is the state with the spinner's next tick due while Claude answers,
// and the command that waits for it.
func (self *AppStatusHelper) Kept(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	if s.Spinning || !answering(s) {
		return s, nil
	}
	s.Spinning = true
	return s, self.c.Tick(spinRate, func(time.Time) tea.Msg { return types.Spun{} })
}

// AfterSpin is the state with the spinner one frame on.
func (self *AppStatusHelper) AfterSpin(s context.ContextTree) context.ContextTree {
	s.Spinning = false
	s.Spin++
	return s
}

func latestStep(s context.ContextTree) string {
	if s.Review == nil {
		return ""
	}
	for _, t := range s.Review.State.Threads {
		if w := t.Working; w != nil && len(w.Steps) > 0 && presentation.Waiting(t) {
			return w.Steps[len(w.Steps)-1].Words
		}
	}
	return ""
}

func answering(s context.ContextTree) bool {
	if s.Asking > 0 {
		return true
	}
	if s.Review == nil {
		return false
	}
	for _, t := range s.Review.State.Threads {
		if presentation.Waiting(t) {
			return true
		}
	}
	return false
}
