package presentation

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const (
	repoPanels  = 9
	authorWidth = 14
)

// UnitKey names a stack, or a lone pull request, in a repository by its
// bottom pull request.
func UnitKey(repo server.PullRepo, u server.Unit) string {
	return fmt.Sprintf("%s#%d", repo.Name, u.Members[0].Number)
}

// UnitName is a stack's, or a lone pull request's, repository and numbers,
// bottom first.
func UnitName(repo server.PullRepo, u server.Unit) string {
	numbers := make([]string, len(u.Members))
	for i, m := range u.Members {
		numbers[i] = fmt.Sprintf("#%d", m.Number)
	}
	return repo.Name + " " + strings.Join(numbers, " → ")
}

// HomePanels are the home screen's panels: one for the accounts GitHub
// could not read, then one for each repository's open pull requests, the
// first nine numbered; or one that says why there are none.
func HomePanels(tr *i18n.TranslationSet, th theme.Theme, list *server.Pulls, failed string) []HomePanel {
	note := func(key, text string) []HomePanel {
		return []HomePanel{{Title: tr.PullRequestsTitle, Repo: -1, Rows: []SideRow{{Key: key, Text: text, Heading: true}}}}
	}
	switch {
	case list == nil && failed != "":
		return note("failed", th.Bad.Render(tr.CouldNotReadList))
	case list == nil:
		return note("reading", th.Muted.Render(tr.ReadingGitHub))
	case len(list.Accounts) == 0:
		return note("unset", th.Muted.Render(tr.NoAccountsSet))
	}
	var panels []HomePanel
	var unread []SideRow
	for _, a := range list.Accounts {
		if a.Error != "" {
			unread = append(unread, SideRow{Key: "account:" + a.Login, Text: th.Bad.Render("✗") + " " + th.Text.Render(a.Login), Right: th.Muted.Render(tr.NotRead), Item: Item{Kind: "account", Title: a.Login}})
		}
	}
	if len(unread) > 0 {
		panels = append(panels, HomePanel{Title: tr.NotReadFromGitHub, Repo: -1, Rows: unread})
	}
	for r, repo := range list.Repos {
		panel := HomePanel{Title: repo.Name, Repo: r}
		if r < repoPanels {
			panel.Number = r + 1
		}
		widest := 0
		for _, u := range repo.Units {
			for _, m := range u.Members {
				widest = max(widest, len(utils.Itoa(m.Number)))
			}
		}
		for _, u := range repo.Units {
			for i, m := range u.Members {
				if len(m.Asked) > 0 {
					panel.Waiting++
				}
				panel.Rows = append(panel.Rows, pullRow(u, i, widest, th))
			}
		}
		panels = append(panels, panel)
	}
	if len(panels) == 0 {
		return note("none", th.Muted.Render(tr.NoneOpen))
	}
	return panels
}

func pullRow(u server.Unit, i, widest int, th theme.Theme) SideRow {
	m := u.Members[i]
	waiting, loaded, bracket := " ", " ", " "
	if len(m.Asked) > 0 {
		waiting = th.Warning.Render("●")
	}
	if m.Review != "" {
		loaded = th.Accent.Render("◆")
	}
	switch {
	case len(u.Members) == 1:
	case i == 0:
		bracket = th.Muted.Render("┌")
	case i == len(u.Members)-1:
		bracket = th.Muted.Render("└")
	default:
		bracket = th.Muted.Render("│")
	}
	number, title := th.Muted, th.Text
	if m.Draft {
		number, title = th.Faint, th.Faint
	}
	return SideRow{
		Key:   fmt.Sprintf("pull:%d", m.Number),
		Text:  waiting + loaded + " " + bracket + " " + number.Render(fmt.Sprintf("%*s", widest+1, fmt.Sprintf("#%d", m.Number))) + " " + title.Render(m.Title),
		Right: th.Muted.Render(ansi.Truncate(m.Author, authorWidth, "…")),
		Item:  Item{Kind: "pull", Pull: m.Number, URL: m.URL, Title: m.Title},
	}
}
