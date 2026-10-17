package gui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var postOrder = []string{"comment", "approve-with-comments", "approve"}

func (a App) queueMember(pull int) (server.QueueMember, bool) {
	if a.Review == nil || a.Review.State.Queue == nil {
		return server.QueueMember{}, false
	}
	for _, m := range a.Review.State.Queue.Members {
		if m.Pull == pull {
			return m, true
		}
	}
	return server.QueueMember{}, false
}

func (a App) openPosting(pull int) App {
	tr := a.c.Tr
	if a.Review == nil || a.Review.State.Queue == nil {
		return a.say(tr.NoQueueToPost, true)
	}
	if a.Review.State.Queue.Moved != nil {
		return a.say(tr.ReviewMovedNoSend, true)
	}
	if members := a.Review.State.Queue.Members; pull == 0 && len(members) == 1 {
		pull = members[0].Pull
	}
	m, ok := a.queueMember(pull)
	if !ok {
		return a.say(tr.PPosts, true)
	}
	pn := context.PostingContext{Pull: m.Pull}
	for _, action := range postOrder {
		if _, offered := m.Sets[action]; offered {
			pn.Actions = append(pn.Actions, action)
		}
	}
	if len(pn.Actions) == 0 {
		return a.say(utils.ResolvePlaceholderString(tr.PullWasState, map[string]string{"pull": utils.Itoa(m.Pull), "state": m.State}), true)
	}
	a.ContextTree = a.Opened(context.PostingKey)
	a.Posting = pn
	return a
}

func (a App) withoutPosting() App {
	a.ContextTree = a.Closed(context.PostingConfirmationKey).Closed(context.PostingKey)
	return a
}

func (a App) postingKey(name string) (tea.Model, tea.Cmd) {
	pn := a.Posting
	if a.Has(context.PostingConfirmationKey) {
		switch name {
		case "y":
			a = a.withoutPosting()
			m, _ := a.queueMember(pn.Pull)
			action := pn.Actions[pn.Choice]
			client, id, build, set := a.opts.Server, a.Review.ID, a.Review.Page.Build, m.Sets[action].Set
			return a.say(a.c.Tr.CheckingThenSending, false), func() tea.Msg {
				message, state, err := client.Post(id, pn.Pull, action, set, build)
				return types.Posted{Message: message, State: state, Err: err}
			}
		case "n", "esc", "q":
			a.ContextTree = a.Closed(context.PostingConfirmationKey)
		}
		return a, nil
	}
	switch name {
	case "j", "down":
		pn.Choice = min(pn.Choice+1, len(pn.Actions)-1)
	case "k", "up":
		pn.Choice = max(pn.Choice-1, 0)
	case "1", "2", "3":
		if n := int(name[0] - '1'); n < len(pn.Actions) {
			pn.Choice = n
		}
	case "enter":
		a.ContextTree = a.Opened(context.PostingConfirmationKey)
	case "n", "esc", "q":
		return a.withoutPosting(), nil
	}
	a.Posting = pn
	return a, nil
}

func (a App) afterPost(msg types.Posted) App {
	tr := a.c.Tr
	title, words, notice := tr.SentToGitHub, msg.Message, tr.SentQCloses
	if msg.Err != nil {
		title, words, notice = tr.NothingPosted, msg.Err.Error(), tr.NothingPostedQCloses
	} else {
		a = a.withState(msg.State)
	}
	pn := context.MenuContext{Menu: presentation.Menu{Title: title, Rows: []presentation.MenuRow{{Right: strings.ToUpper(words[:1]) + words[1:]}}}}
	return a.withMenu(pn).say(notice, msg.Err != nil)
}

func (a App) popupWidth() int {
	return min(a.Width-4, max(a.Width*3/5, 60))
}

func (a App) postingBox() []string {
	pn := a.Posting
	m, _ := a.queueMember(pn.Pull)
	return presentation.PostingBox(a.c.Tr, a.c.Theme, a.Review.State.Queue.Repo, pn.Pull, m, pn.Actions, pn.Choice, a.Has(context.PostingConfirmationKey), a.popupWidth())
}

func (a App) postingHints() [][2]string {
	pn, tr := a.Posting, a.c.Tr
	if a.Has(context.PostingConfirmationKey) {
		yes := utils.ResolvePlaceholderString(tr.YesAction, map[string]string{"action": strings.ToLower(presentation.PostLabels(tr)[pn.Actions[pn.Choice]])})
		return [][2]string{{"y", yes}, {"n", tr.Cancel}}
	}
	return [][2]string{{"j k 1 2 3", tr.Choose}, {"Enter", tr.ShowWhatItSends}, {"Esc", tr.Cancel}}
}
