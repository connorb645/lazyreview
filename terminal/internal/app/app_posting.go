package app

import (
	"fmt"
	"path"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"review-desk/terminal/internal/server"
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

func (a App) openPosting() App {
	if a.review == nil || a.review.state.Queue == nil {
		return a.say("only a pull request's review has a queue to post", true)
	}
	if a.review.state.Queue.Moved != nil {
		return a.say("this review moved, so nothing can be sent from here", true)
	}
	pull := 0
	if v := a.rv.view; v.cursor < len(v.items) {
		pull = v.items[v.cursor].pull
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
	pn := panel{title: title}
	for _, l := range wrap(strings.ToUpper(words[:1])+words[1:], a.width-6) {
		pn.rows = append(pn.rows, panelRow{right: l})
	}
	a.panel = &pn
	return a.say(notice, msg.err != nil)
}

func (a App) postingLines(width int) []string {
	p, pn := a.p, *a.posting
	m, _ := a.queueMember(pn.pull)
	inner := width - 4
	frame := func(text string) string {
		return p.Muted.Render("│ ") + fit(text, inner) + p.Muted.Render(" │")
	}
	out := []string{p.Muted.Render("╭" + strings.Repeat("─", width-2) + "╮"), frame(p.Bold.Render(fmt.Sprintf("Post to %s#%d", a.review.state.Queue.Repo, pn.pull)))}
	for i, action := range pn.actions {
		label := postLabels[action]
		if action != "approve" {
			label += fmt.Sprintf(" (%d)", len(m.Comments))
		}
		mark, style := "  ", p.Ink
		if i == pn.choice {
			mark, style = p.Accent.Render("▸ "), p.Bold
		}
		out = append(out, frame(mark+p.Key.Render(fmt.Sprintf("%d", i+1))+"  "+style.Render(label)))
	}
	if !pn.confirming {
		return append(out, p.Muted.Render("╰"+strings.Repeat("─", width-2)+"╯"))
	}
	action := pn.actions[pn.choice]
	out = append(out, p.Muted.Render("├"+strings.Repeat("─", width-2)+"┤"))
	for _, l := range wrap(m.Sets[action].Words, inner) {
		out = append(out, frame(p.Ink.Render(l)))
	}
	if action != "approve" {
		place := 0
		for _, c := range m.Comments {
			place = max(place, ansi.StringWidth(commentPlace(c)))
		}
		for i, c := range m.Comments {
			first, _, _ := strings.Cut(c.Body, "\n")
			text := fmt.Sprintf("%3d  %-*s  %s", i+1, place, commentPlace(c), first)
			out = append(out, frame(p.Ink.Render(text)))
		}
	}
	return append(out, p.Muted.Render("╰"+strings.Repeat("─", width-2)+"╯"))
}

func commentPlace(c server.Comment) string {
	return path.Base(c.Path) + ", " + linesWords(c.Side, c.Line, c.StartLine, c.StartSide)
}

func (a App) postingHints() [][2]string {
	pn := a.posting
	if pn.confirming {
		return [][2]string{{"y", "yes, " + strings.ToLower(postLabels[pn.actions[pn.choice]])}, {"n", "cancel"}}
	}
	return [][2]string{{"j k 1 2 3", "choose"}, {"Enter", "shows what it sends"}, {"Esc", "cancel"}}
}
