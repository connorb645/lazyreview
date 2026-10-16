package app

import (
	"fmt"
	"path"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/server"
)

var (
	postOrder  = []string{"comment", "approve-with-comments", "approve"}
	postLabels = map[string]string{"comment": "Add Comments", "approve-with-comments": "Approve With Comments", "approve": "Approve"}
)

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
	if a.review == nil || a.review.state.Queue == nil {
		return a.say("only a pull request's review has a queue to post", true)
	}
	if a.review.state.Queue.Moved != nil {
		return a.say("this review moved, so nothing can be sent from here", true)
	}
	if members := a.review.state.Queue.Members; pull == 0 && len(members) == 1 {
		pull = members[0].Pull
	}
	m, ok := a.queueMember(pull)
	if !ok {
		return a.say("P posts for the pull request at the cursor; move to it, or to one of its comments", true)
	}
	pn := posting{pull: m.Pull}
	for _, action := range postOrder {
		if _, offered := m.Sets[action]; offered {
			pn.actions = append(pn.actions, action)
		}
	}
	if len(pn.actions) == 0 {
		return a.say(fmt.Sprintf("#%d was %s, so nothing can be sent to it", m.Pull, m.State), true)
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
			return a.say("Checking the pull request with GitHub, then sending…", false), func() tea.Msg {
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
	title, words, notice := "Sent to GitHub", msg.message, "Sent; q closes the server's answer."
	if msg.err != nil {
		title, words, notice = "Nothing was posted", msg.err.Error(), "Nothing was posted; q closes why."
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
	p, pn := a.p, *a.posting
	m, _ := a.queueMember(pn.pull)
	w := a.popupWidth()
	inner := w - 2
	var out []string
	for i, action := range pn.actions {
		label := postLabels[action]
		if action != "approve" {
			label += fmt.Sprintf(" (%d)", len(m.Comments))
		}
		row := " " + p.Key.Render(fmt.Sprintf("%d", i+1)) + "  " + p.Ink.Render(label)
		if i == pn.choice {
			row = p.highlight(row, inner, !pn.confirming)
		}
		out = append(out, row)
	}
	footer := fmt.Sprintf("%d of %d", pn.choice+1, len(pn.actions))
	if pn.confirming {
		action := pn.actions[pn.choice]
		out = append(out, p.Border.Render(strings.Repeat("─", inner)))
		for _, l := range wrap(m.Sets[action].Words, inner-2) {
			out = append(out, " "+p.Ink.Render(l))
		}
		if action != "approve" {
			place := 0
			for _, c := range m.Comments {
				place = max(place, ansi.StringWidth(commentPlace(c)))
			}
			out = append(out, "")
			for i, c := range m.Comments {
				first, _, _ := strings.Cut(c.Body, "\n")
				out = append(out, " "+p.Muted.Render(fmt.Sprintf("%2d  %-*s", i+1, place, commentPlace(c)))+"  "+p.Ink.Render(first))
			}
		}
		footer = ""
	}
	return framed(out, extent{}, w, len(out)+2, fmt.Sprintf("Post to %s#%d", a.review.state.Queue.Repo, pn.pull), "", footer, true, p)
}

func commentPlace(c server.Comment) string {
	return path.Base(c.Path) + ", " + linesWords(c.Side, c.Line, c.StartLine, c.StartSide)
}

func (a App) postingHints() [][2]string {
	pn := a.posting
	if pn.confirming {
		return [][2]string{{"y", "Yes, " + strings.ToLower(postLabels[pn.actions[pn.choice]])}, {"n", "Cancel"}}
	}
	return [][2]string{{"j k 1 2 3", "Choose"}, {"Enter", "Show what it sends"}, {"Esc", "Cancel"}}
}
