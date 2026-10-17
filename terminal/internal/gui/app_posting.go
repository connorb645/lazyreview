package gui

import (
	"fmt"
	"path"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var postOrder = []string{"comment", "approve-with-comments", "approve"}

func postLabels(tr *i18n.TranslationSet) map[string]string {
	return map[string]string{"comment": tr.AddComments, "approve-with-comments": tr.ApproveWithComments, "approve": tr.Approve}
}

func (a App) queueMember(pull int) (server.QueueMember, bool) {
	if a.review == nil || a.review.state.Queue == nil {
		return server.QueueMember{}, false
	}
	for _, m := range a.review.state.Queue.Members {
		if m.Pull == pull {
			return m, true
		}
	}
	return server.QueueMember{}, false
}

func (a App) openPosting(pull int) App {
	tr := a.c.Tr
	if a.review == nil || a.review.state.Queue == nil {
		return a.say(tr.NoQueueToPost, true)
	}
	if a.review.state.Queue.Moved != nil {
		return a.say(tr.ReviewMovedNoSend, true)
	}
	if members := a.review.state.Queue.Members; pull == 0 && len(members) == 1 {
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
			client, id, build, set := a.opts.Server, a.review.id, a.review.page.Build, m.Sets[action].Set
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
	pn := menu{title: title, rows: []menuRow{{right: strings.ToUpper(words[:1]) + words[1:]}}}
	a.menu = &pn
	return a.say(notice, msg.err != nil)
}

func (a App) popupWidth() int {
	return min(a.width-4, max(a.width*3/5, 60))
}

func (a App) postingBox() []string {
	p, pn, tr := a.p, *a.posting, a.c.Tr
	m, _ := a.queueMember(pn.pull)
	w := a.popupWidth()
	inner := w - 2
	var out []string
	for i, action := range pn.actions {
		label := postLabels(tr)[action]
		if action != "approve" {
			label += fmt.Sprintf(" (%d)", len(m.Comments))
		}
		row := " " + p.Key.Render(fmt.Sprintf("%d", i+1)) + "  " + p.Ink.Render(label)
		if i == pn.choice {
			row = p.highlight(row, inner, !pn.confirming)
		}
		out = append(out, row)
	}
	footer := ofTotal(tr, pn.choice+1, len(pn.actions))
	if pn.confirming {
		action := pn.actions[pn.choice]
		out = append(out, p.Border.Render(strings.Repeat("─", inner)))
		for _, l := range utils.Wrap(m.Sets[action].Words, inner-2) {
			out = append(out, " "+p.Ink.Render(l))
		}
		if action != "approve" {
			place := 0
			for _, c := range m.Comments {
				place = max(place, ansi.StringWidth(commentPlace(tr, c)))
			}
			out = append(out, "")
			for i, c := range m.Comments {
				first, _, _ := strings.Cut(c.Body, "\n")
				out = append(out, " "+p.Muted.Render(fmt.Sprintf("%2d  %-*s", i+1, place, commentPlace(tr, c)))+"  "+p.Ink.Render(first))
			}
		}
		footer = ""
	}
	title := utils.ResolvePlaceholderString(tr.PostTo, map[string]string{"repo": a.review.state.Queue.Repo, "pull": utils.Itoa(pn.pull)})
	return framed(out, extent{}, w, len(out)+2, title, "", footer, true, p)
}

func commentPlace(tr *i18n.TranslationSet, c server.Comment) string {
	return path.Base(c.Path) + ", " + linesWords(tr, c.Side, c.Line, c.StartLine, c.StartSide)
}

func (a App) postingHints() [][2]string {
	pn, tr := a.posting, a.c.Tr
	if pn.confirming {
		yes := utils.ResolvePlaceholderString(tr.YesAction, map[string]string{"action": strings.ToLower(postLabels(tr)[pn.actions[pn.choice]])})
		return [][2]string{{"y", yes}, {"n", tr.Cancel}}
	}
	return [][2]string{{"j k 1 2 3", tr.Choose}, {"Enter", tr.ShowWhatItSends}, {"Esc", tr.Cancel}}
}
