package helpers

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// PromptHelper asks for a command or a search in the bar, and does what
// was typed.
type PromptHelper struct {
	c *HelperCommon
}

// Ask is the state with the bar asking for what follows prefix: ":" for a
// command, "/" or "?" for a search.
func (self *PromptHelper) Ask(s context.ContextTree, prefix string) context.ContextTree {
	in := textinput.New()
	in.Prompt = prefix
	in.Cursor.SetMode(cursor.CursorStatic)
	in.Focus()
	s = s.Opened(context.PromptKey)
	s.Prompt = context.PromptContext{Input: in, Kind: prefix}
	return s
}

// Search is the state with the bar asking for a search, forward or back.
func (self *PromptHelper) Search(s context.ContextTree, back bool) context.ContextTree {
	s.SearchBack = back
	return self.Ask(s, map[bool]string{false: "/", true: "?"}[back])
}

// Typed is the state with a message, such as a key, given to the bar.
func (self *PromptHelper) Typed(s context.ContextTree, msg tea.Msg) (context.ContextTree, tea.Cmd) {
	var cmd tea.Cmd
	s.Prompt.Input, cmd = s.Prompt.Input.Update(msg)
	return s, cmd
}

// Submitted is the state once what the bar asked for is entered: the
// search done, in the diagram or the diff, or the command.
func (self *PromptHelper) Submitted(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	h := self.c.Helpers
	typed := strings.TrimSpace(s.Prompt.Input.Value())
	kind := s.Prompt.Kind
	s = s.Closed(context.PromptKey)
	switch {
	case s.Screen() == context.ScreenDiagram && (kind == "/" || kind == "?"):
		s.Diagram.Query = typed
		next, cmd := h.Diagram.FindBox(s, s.SearchBack)
		return h.Refresh.Mirrored(next), cmd
	case kind == "/" || kind == "?":
		s.Diff.Query = typed
		next, cmd := h.Diff.MatchNext(s, s.SearchBack)
		return h.Refresh.Mirrored(next), cmd
	}
	return self.command(s, typed)
}

func (self *PromptHelper) command(s context.ContextTree, typed string) (context.ContextTree, tea.Cmd) {
	h := self.c.Helpers
	switch typed {
	case "":
		return s, nil
	case "q", "q!", "quit", "qa", "qa!":
		return s, tea.Quit
	case "refresh":
		return h.Review.Refresh(s)
	}
	if n, err := strconv.Atoi(typed); err == nil && s.Screen() == context.ScreenDiff && s.Review != nil {
		s = h.Jump.Leave(s)
		s = s.Focused(context.MainPanel, s.Driving())
		next, cmd := h.Diff.ToLine(s, n)
		return h.Refresh.Mirrored(next), cmd
	}
	if arg, ok := strings.CutPrefix(typed, "pull "); ok {
		return h.Filter.PullCommand(s, strings.TrimSpace(arg)), nil
	}
	return s.Said(utils.ResolvePlaceholderString(self.c.Tr.NotACommand, map[string]string{"command": typed}), true), nil
}
