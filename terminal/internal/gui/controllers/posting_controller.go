package controllers

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers/helpers"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// PostingController binds the keys of posting's choice: the actions
// offered for a pull request's queued comments.
type PostingController struct {
	baseController
	c *helpers.HelperCommon
}

// NewPostingController binds the keys of posting's choice.
func NewPostingController(c *helpers.HelperCommon) *PostingController {
	return &PostingController{c: c}
}

// Context is posting's.
func (self *PostingController) Context() context.ContextKey {
	return context.PostingKey
}

// Bindings are posting's keys: choosing an action, by moving or by its
// number, showing what it sends, and closing.
func (self *PostingController) Bindings() []Binding {
	tr, kb := self.c.Tr, self.c.Config.Keybinding
	bindings := []Binding{
		{Action: "down", Keys: kb.Posting.Down, Handler: self.chosen(func(p context.PostingContext) int { return min(p.Choice+1, len(p.Actions)-1) }), Description: tr.Down},
		{Action: "up", Keys: kb.Posting.Up, Handler: self.chosen(func(p context.PostingContext) int { return max(p.Choice-1, 0) }), Description: tr.Up},
	}
	for n := 1; n <= 3; n++ {
		bindings = append(bindings, Binding{
			Action: "choose " + utils.Itoa(n),
			Keys:   numbered(kb.Posting.Choose, n-1),
			Handler: self.chosen(func(p context.PostingContext) int {
				if n-1 < len(p.Actions) {
					return n - 1
				}
				return p.Choice
			}),
			Description: utils.ResolvePlaceholderString(tr.ChooseNumber, map[string]string{"number": utils.Itoa(n)}),
		})
	}
	return append(bindings,
		Binding{Action: "open", Keys: kb.Posting.Open, Handler: stateOnly(self.confirming), Description: tr.ShowWhatItSends},
		Binding{Action: "close", Keys: kb.Posting.Close, Handler: stateOnly(context.ContextTree.WithoutPosting), Description: tr.Cancel},
	)
}

func (self *PostingController) chosen(choice func(p context.PostingContext) int) Handler {
	return func(s context.ContextTree, _ Count) (context.ContextTree, tea.Cmd) {
		s.Posting.Choice = choice(s.Posting)
		return s, nil
	}
}

func (self *PostingController) confirming(s context.ContextTree) context.ContextTree {
	return s.Opened(context.PostingConfirmationKey)
}

// Unbound takes every other key and does nothing with it.
func (self *PostingController) Unbound(s context.ContextTree, _ string, _ tea.KeyMsg) (context.ContextTree, tea.Cmd, bool) {
	return s, nil, false
}

// Hints are what the bar shows of posting's keys: choosing only among
// more than one action, by the numbers there are.
func (self *PostingController) Hints(s context.ContextTree, labels presentation.KeyLabels) [][2]string {
	tr := self.c.Tr
	hints := [][2]string{{labels.Of("open"), tr.ShowWhatItSends}, {labels.Of("close"), tr.Cancel}}
	if len(s.Posting.Actions) < 2 {
		return hints
	}
	choose := []string{"down", "up"}
	for n := 1; n <= min(len(s.Posting.Actions), 3); n++ {
		choose = append(choose, "choose "+utils.Itoa(n))
	}
	return append([][2]string{{labels.Of(choose...), tr.Choose}}, hints...)
}
