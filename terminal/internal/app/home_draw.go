package app

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"review-desk/terminal/internal/server"
)

var htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)

var settingsSample = []string{
	`{"accounts": [`,
	`  {"login": "your-login"},`,
	`  {"login": "your-work-login",`,
	`   "teams": [`,
	`     {"name": "your-org/your-team",`,
	`      "since": "2026-10-01"}`,
	`   ]}`,
	`]}`,
}

func (a App) homePanelBox(i int, r rect, focused bool) []string {
	panel, p := a.home.panels[i], a.p
	content, place := sideContent(panel.rows, r.inner(), focused, p)
	if panel.rows.cursor < 0 {
		place = ""
	}
	title := panel.title
	if panel.number > 0 {
		title = panelTitle(panel.number, title)
	}
	waiting := ""
	if panel.waiting > 0 {
		waiting = " " + p.Warn.Render(fmt.Sprintf("● %d", panel.waiting))
	}
	return framed(content, extent{}, r.w, r.h, title, waiting, place, focused, p)
}

func (a App) homeMainBox(r rect) []string {
	h, p, width := a.home, a.p, r.inner().w
	b := &builder{width: width, p: p}
	title := "Pull requests"
	repo, u, m, chosen := h.pull()
	row, _ := h.selected()
	switch {
	case h.list == nil && h.failed != "":
		b.text("")
		b.prose(" ", p.Bad, "Could not read the open pull requests: "+h.failed)
		b.text("")
		b.prose(" ", p.Muted, "R tries again; :q quits.")
	case h.list == nil:
		b.text("")
		b.prose(" ", p.Muted, "Reading the open pull requests from GitHub…")
	case len(h.list.Accounts) == 0:
		title = "Accounts"
		b.text("")
		b.prose(" ", p.Ink, "No GitHub accounts are set, so there is nothing to list yet. Name them in "+fromHome(h.list.Settings, a.opts.Home)+":")
		b.text("")
		for _, l := range settingsSample {
			b.text("   " + p.Accent.Render(l))
		}
		b.text("")
		b.prose(" ", p.Muted, "Each account is read by its own token, so each must be signed in to gh. A review asked of a team counts as asked of you, for pull requests opened on or after its since date. R reads the settings again.")
	case chosen:
		title = fmt.Sprintf("%s #%d", repo.Name, m.Number)
		a.pullLines(b, repo, u, m)
	case row.it.kind == "account":
		title = row.it.title
		b.text("")
		b.prose(" ", p.Ink, "GitHub could not be read as "+row.it.title+":")
		for _, read := range h.list.Accounts {
			if read.Login == row.it.title {
				b.prose(" ", p.Bad, read.Error)
			}
		}
		b.text("")
		b.prose(" ", p.Muted, "Its pull requests are left out until GitHub answers. R tries again.")
	default:
		logins := make([]string, len(h.list.Accounts))
		for i, read := range h.list.Accounts {
			logins[i] = read.Login
		}
		b.text("")
		b.prose(" ", p.Ink, "No open pull request involves "+listed(logins)+": none asks for a review from them or their teams, none is theirs, and none has their comments.")
		b.text("")
		b.prose(" ", p.Muted, "R reads GitHub again.")
	}
	content := make([]string, len(b.lines))
	for i, l := range b.lines {
		content[i] = l.text
	}
	return framed(content, extent{}, r.w, r.h, panelTitle(mainPanel, title), "", "", false, p)
}

func (a App) pullLines(b *builder, repo server.PullRepo, u server.Unit, m server.OpenPull) {
	p := a.p
	b.prose(" ", p.Bold, m.Title)
	b.parts(" ", p.Muted.Render(fmt.Sprintf("#%d by %s", m.Number, m.Author)), p.Muted.Render(m.Head+" → "+m.Base))
	b.text("")
	if len(m.Asked) > 0 {
		marked(b, p.Warn.Render("●"), p.Ink, "Waiting on your review, asked of "+listed(m.Asked)+".")
	}
	if m.Review != "" {
		marked(b, p.Accent.Render("◆"), p.Ink, "Loaded in Review Desk.")
	} else {
		marked(b, " ", p.Muted, "Not loaded in Review Desk yet.")
	}
	if m.Yours {
		marked(b, " ", p.Muted, "You opened it.")
	}
	if m.Commented {
		marked(b, " ", p.Muted, "You have commented on it.")
	}
	if m.Draft {
		marked(b, " ", p.Muted, "A draft, not ready for review yet.")
	}
	b.text("")
	switch {
	case a.home.loading == unitKey(repo, u):
		b.prose(" ", p.Accent, "Loading from GitHub…")
	case u.Review != "" && len(u.Members) > 1:
		b.prose(" ", p.Ink, "Enter opens the stack's review.")
	case u.Review != "":
		b.prose(" ", p.Ink, "Enter opens its review.")
	case len(u.Members) > 1:
		b.prose(" ", p.Ink, "Enter loads the stack, "+strings.TrimPrefix(unitName(repo, u), repo.Name+" ")+", as "+repo.Account+", then opens it.")
	default:
		b.prose(" ", p.Ink, "Enter loads it as "+repo.Account+", then opens it.")
	}
	if len(u.Members) > 1 {
		b.text("", " "+p.Muted.Render("The stack, bottom first:"))
		for _, other := range u.Members {
			mark, style := "  ", p.Muted
			if other.Number == m.Number {
				mark, style = p.Accent.Render("▸ "), p.Ink
			}
			b.text(" " + mark + style.Render(fmt.Sprintf("#%d %s", other.Number, other.Title)))
		}
	}
	b.text("")
	description := strings.TrimSpace(htmlComment.ReplaceAllString(m.Description, ""))
	if description == "" {
		b.text(" " + p.Muted.Render("No description."))
		return
	}
	for _, l := range withoutMargin(a.md.lines(description, b.width-2)) {
		b.text(" " + l)
	}
}

func marked(b *builder, mark string, style lipgloss.Style, text string) {
	for i, l := range wrap(text, b.width-4) {
		if i == 0 {
			b.text(" " + mark + " " + style.Render(l))
			continue
		}
		b.text("   " + style.Render(l))
	}
}

func fromHome(path, home string) string {
	if rest, ok := strings.CutPrefix(path, home+"/"); ok && home != "" {
		return "~/" + rest
	}
	return path
}

func listed(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
