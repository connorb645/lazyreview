package gui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)

func (a App) homePanelBox(i int, r rect, focused bool) []string {
	panel, p := a.home.panels[i], a.p
	content, place := sideContent(a.c.Tr, panel.rows, r.inner(), focused, p)
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
	h, p, width, tr := a.home, a.p, r.inner().w, a.c.Tr
	b := &builder{width: width, tr: tr, p: p}
	title := tr.PullRequestsTitle
	repo, u, m, chosen := h.pull()
	row, _ := h.selected()
	switch {
	case h.list == nil && h.failed != "":
		b.text("")
		b.prose(" ", p.Bad, tr.CouldNotReadPulls+": "+h.failed)
		b.text("")
		b.prose(" ", p.Muted, tr.TryAgainOrQuit)
	case h.list == nil:
		b.text("")
		b.prose(" ", p.Muted, tr.ReadingPulls)
	case len(h.list.Accounts) == 0:
		title = tr.AccountsTitle
		b.text("")
		b.prose(" ", p.Ink, utils.ResolvePlaceholderString(tr.NameAccountsIn, map[string]string{"path": fromHome(h.list.Settings, a.opts.Home)}))
		b.text("")
		for _, l := range strings.Split(tr.SettingsSample, "\n") {
			b.text("   " + p.Accent.Render(l))
		}
		b.text("")
		b.prose(" ", p.Muted, tr.EachAccount)
	case chosen:
		title = fmt.Sprintf("%s #%d", repo.Name, m.Number)
		a.pullLines(b, repo, u, m)
	case row.it.kind == "account":
		title = row.it.title
		b.text("")
		b.prose(" ", p.Ink, utils.ResolvePlaceholderString(tr.CouldNotReadAs, map[string]string{"login": row.it.title}))
		for _, read := range h.list.Accounts {
			if read.Login == row.it.title {
				b.prose(" ", p.Bad, read.Error)
			}
		}
		b.text("")
		b.prose(" ", p.Muted, tr.LeftOutUntil)
	default:
		logins := make([]string, len(h.list.Accounts))
		for i, read := range h.list.Accounts {
			logins[i] = read.Login
		}
		b.text("")
		b.prose(" ", p.Ink, utils.ResolvePlaceholderString(tr.NothingInvolves, map[string]string{"logins": listed(tr, logins)}))
		b.text("")
		b.prose(" ", p.Muted, tr.ReadsGitHubAgain)
	}
	content := make([]string, len(b.lines))
	for i, l := range b.lines {
		content[i] = l.text
	}
	return framed(content, extent{}, r.w, r.h, panelTitle(mainPanel, title), "", "", false, p)
}

func (a App) pullLines(b *builder, repo server.PullRepo, u server.Unit, m server.OpenPull) {
	p, tr := a.p, a.c.Tr
	b.prose(" ", p.Bold, m.Title)
	b.parts(" ", p.Muted.Render(utils.ResolvePlaceholderString(tr.PullBy, map[string]string{"pull": utils.Itoa(m.Number), "author": m.Author})), p.Muted.Render(m.Head+" → "+m.Base))
	b.text("")
	if len(m.Asked) > 0 {
		marked(b, p.Warn.Render("●"), p.Ink, utils.ResolvePlaceholderString(tr.WaitingOnYou, map[string]string{"who": listed(tr, m.Asked)}))
	}
	if m.Review != "" {
		marked(b, p.Accent.Render("◆"), p.Ink, tr.LoadedInLazyreview)
	} else {
		marked(b, " ", p.Muted, tr.NotLoadedYet)
	}
	if m.Yours {
		marked(b, " ", p.Muted, tr.YouOpenedIt)
	}
	if m.Commented {
		marked(b, " ", p.Muted, tr.YouCommented)
	}
	if m.Draft {
		marked(b, " ", p.Muted, tr.ADraft)
	}
	b.text("")
	switch {
	case a.home.loading == unitKey(repo, u):
		b.prose(" ", p.Accent, tr.LoadingFromGitHubNow)
	case u.Review != "" && len(u.Members) > 1:
		b.prose(" ", p.Ink, tr.EnterOpensStack)
	case u.Review != "":
		b.prose(" ", p.Ink, tr.EnterOpensReview)
	case len(u.Members) > 1:
		pulls := strings.TrimPrefix(unitName(repo, u), repo.Name+" ")
		b.prose(" ", p.Ink, utils.ResolvePlaceholderString(tr.EnterLoadsStack, map[string]string{"pulls": pulls, "account": repo.Account}))
	default:
		b.prose(" ", p.Ink, utils.ResolvePlaceholderString(tr.EnterLoadsIt, map[string]string{"account": repo.Account}))
	}
	if len(u.Members) > 1 {
		b.text("", " "+p.Muted.Render(tr.StackBottomFirst))
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
		b.text(" " + p.Muted.Render(tr.NoDescription))
		return
	}
	for _, l := range withoutMargin(a.md.lines(description, b.width-2)) {
		b.text(" " + l)
	}
}

func marked(b *builder, mark string, style lipgloss.Style, text string) {
	for i, l := range utils.Wrap(text, b.width-4) {
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

func listed(tr *i18n.TranslationSet, names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return utils.ResolvePlaceholderString(tr.ListAnd, map[string]string{"list": strings.Join(names[:len(names)-1], ", "), "last": names[len(names)-1]})
}
