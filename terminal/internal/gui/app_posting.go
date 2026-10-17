package gui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var postOrder = []string{"comment", "approve-with-comments", "approve"}

func (a App) queueMember(pull int) (server.QueueMember, bool) {
	if a.review == nil || a.review.State.Queue == nil {
		return server.QueueMember{}, false
	}
	for _, m := range a.review.State.Queue.Members {
		if m.Pull == pull {
			return m, true
		}
	}
	return server.QueueMember{}, false
}

func (a App) openPosting(pull int) App {
	tr := a.c.Tr
	if a.review == nil || a.review.State.Queue == nil {
		return a.say(tr.NoQueueToPost, true)
	}
	if a.review.State.Queue.Moved != nil {
		return a.say(tr.ReviewMovedNoSend, true)
	}
	if members := a.review.State.Queue.Members; pull == 0 && len(members) == 1 {
		pull = members[0].Pull
	}
	m, ok := a.queueMember(pull)
	if !ok {
		return a.say(tr.PPosts, true)
	}
	pn := posting{pull: m.Pull}
	for _, action := range postOrder {
		if _, offered := m.Sets[action]; offered {
			pn.actions = append(pn.actions, action)
		}
	}
	if len(pn.actions) == 0 {
		return a.say(utils.ResolvePlaceholderString(tr.PullWasState, map[string]string{"pull": utils.Itoa(m.Pull), "state": m.State}), true)
	}
	a.posting = &pn
	return a
}

func (a App) postingKey(name string) (tea.Model, tea.Cmd) {
	pn := *a.posting
	if pn.confirming {
		switch name {
		case "y":
			a.posting = nil
			m, _ := a.queueMember(pn.pull)
			action := pn.actions[pn.choice]
			client, id, build, set := a.opts.Server, a.review.ID, a.review.Page.Build, m.Sets[action].Set
			return a.say(a.c.Tr.CheckingThenSending, false), func() tea.Msg {
				message, state, err := client.Post(id, pn.pull, action, set, build)
				return posted{message: message, state: state, err: err}
			}
		case "n", "esc", "q":
			pn.confirming = false
		}
		a.posting = &pn
		return a, nil
	}
	switch name {
	case "j", "down":
		pn.choice = min(pn.choice+1, len(pn.actions)-1)
	case "k", "up":
		pn.choice = max(pn.choice-1, 0)
	case "1", "2", "3":
		if n := int(name[0] - '1'); n < len(pn.actions) {
			pn.choice = n
		}
	case "enter":
		pn.confirming = true
	case "n", "esc", "q":
		a.posting = nil
		return a, nil
	}
	a.posting = &pn
	return a, nil
}

func (a App) afterPost(msg posted) App {
	tr := a.c.Tr
	title, words, notice := tr.SentToGitHub, msg.message, tr.SentQCloses
	if msg.err != nil {
		title, words, notice = tr.NothingPosted, msg.err.Error(), tr.NothingPostedQCloses
	} else {
		a = a.withState(msg.state)
	}
	pn := menu{Menu: presentation.Menu{Title: title, Rows: []presentation.MenuRow{{Right: strings.ToUpper(words[:1]) + words[1:]}}}}
	a.menu = &pn
	return a.say(notice, msg.err != nil)
}

func (a App) popupWidth() int {
	return min(a.width-4, max(a.width*3/5, 60))
}

func (a App) postingBox() []string {
	pn := *a.posting
	m, _ := a.queueMember(pn.pull)
	return presentation.PostingBox(a.c.Tr, a.c.Theme, a.review.State.Queue.Repo, pn.pull, m, pn.actions, pn.choice, pn.confirming, a.popupWidth())
}

func (a App) postingHints() [][2]string {
	pn, tr := a.posting, a.c.Tr
	if pn.confirming {
		yes := utils.ResolvePlaceholderString(tr.YesAction, map[string]string{"action": strings.ToLower(presentation.PostLabels(tr)[pn.actions[pn.choice]])})
		return [][2]string{{"y", yes}, {"n", tr.Cancel}}
	}
	return [][2]string{{"j k 1 2 3", tr.Choose}, {"Enter", tr.ShowWhatItSends}, {"Esc", tr.Cancel}}
}
