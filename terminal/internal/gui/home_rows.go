package gui

import (
	"fmt"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const (
	repoPanels  = 9
	authorWidth = 14
)

func unitKey(repo server.PullRepo, u server.Unit) string {
	return fmt.Sprintf("%s#%d", repo.Name, u.Members[0].Number)
}

func homePanels(h homeView, p palette) []homePanel {
	note := func(title, key, text string) []homePanel {
		return []homePanel{{title: title, repo: -1, rows: sidePanel{}.reset([]sideRow{{key: key, text: text, heading: true}})}}
	}
	switch {
	case h.list == nil && h.failed != "":
		return note("Pull requests", "failed", p.Bad.Render("Could not read the list"))
	case h.list == nil:
		return note("Pull requests", "reading", p.Muted.Render("Reading GitHub…"))
	case len(h.list.Accounts) == 0:
		return note("Pull requests", "unset", p.Muted.Render("No accounts are set"))
	}
	var panels []homePanel
	var unread []sideRow
	for _, a := range h.list.Accounts {
		if a.Error != "" {
			unread = append(unread, sideRow{key: "account:" + a.Login, text: p.Bad.Render("✗") + " " + p.Ink.Render(a.Login), right: p.Muted.Render("not read"), it: item{kind: "account", title: a.Login}})
		}
	}
	if len(unread) > 0 {
		panels = append(panels, homePanel{title: "Not read from GitHub", repo: -1, rows: sidePanel{}.reset(unread)})
	}
	for r, repo := range h.list.Repos {
		panel := homePanel{title: repo.Name, repo: r}
		if r < repoPanels {
			panel.number = r + 1
		}
		widest := 0
		for _, u := range repo.Units {
			for _, m := range u.Members {
				widest = max(widest, len(utils.Itoa(m.Number)))
			}
		}
		var rows []sideRow
		for _, u := range repo.Units {
			for i, m := range u.Members {
				if len(m.Asked) > 0 {
					panel.waiting++
				}
				rows = append(rows, pullRow(u, i, widest, p))
			}
		}
		panel.rows = sidePanel{}.reset(rows)
		panels = append(panels, panel)
	}
	if len(panels) == 0 {
		return note("Pull requests", "none", p.Muted.Render("None open"))
	}
	return panels
}

func pullRow(u server.Unit, i, widest int, p palette) sideRow {
	m := u.Members[i]
	waiting, loaded, bracket := " ", " ", " "
	if len(m.Asked) > 0 {
		waiting = p.Warn.Render("●")
	}
	if m.Review != "" {
		loaded = p.Accent.Render("◆")
	}
	switch {
	case len(u.Members) == 1:
	case i == 0:
		bracket = p.Muted.Render("┌")
	case i == len(u.Members)-1:
		bracket = p.Muted.Render("└")
	default:
		bracket = p.Muted.Render("│")
	}
	number, title := p.Muted, p.Ink
	if m.Draft {
		number, title = p.Faint, p.Faint
	}
	return sideRow{
		key:   fmt.Sprintf("pull:%d", m.Number),
		text:  waiting + loaded + " " + bracket + " " + number.Render(fmt.Sprintf("%*s", widest+1, fmt.Sprintf("#%d", m.Number))) + " " + title.Render(m.Title),
		right: p.Muted.Render(ansi.Truncate(m.Author, authorWidth, "…")),
		it:    item{kind: "pull", pull: m.Number, url: m.URL, title: m.Title},
	}
}
