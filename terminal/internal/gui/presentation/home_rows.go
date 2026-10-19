package presentation

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const (
	repoPanels  = 9
	authorWidth = 14
)

// UnitKey names a stack, or a lone pull request, in a repository by its
// bottom pull request.
func UnitKey(repo models.PullRepo, u models.Unit) string {
	return fmt.Sprintf("%s#%d", repo.Name, u.Members[0].Number)
}

// UnitName is a stack's, or a lone pull request's, repository and numbers,
// bottom first.
func UnitName(repo models.PullRepo, u models.Unit) string {
	numbers := make([]string, len(u.Members))
	for i, m := range u.Members {
		numbers[i] = fmt.Sprintf("#%d", m.Number)
	}
	return repo.Name + " " + strings.Join(numbers, " → ")
}

// HomePanels are the home screen's panels: one for the accounts GitHub
// could not read, then one for each repository's open pull requests, the
// first nine numbered; or one that says why there are none. A repository the
// config hides has no panel, unless showHidden, when its panel comes after
// the others, dim.
func HomePanels(tr *i18n.TranslationSet, th theme.Theme, list *models.Pulls, failed string, showHidden bool) []HomePanel {
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
	var order []int
	for _, hidden := range []bool{false, true} {
		for r, repo := range list.Repos {
			if (len(repo.HiddenBy) > 0) == hidden && (!hidden || showHidden) {
				order = append(order, r)
			}
		}
	}
	for at, r := range order {
		repo := list.Repos[r]
		panel := HomePanel{Title: repo.Name, Repo: r, Hidden: len(repo.HiddenBy) > 0}
		if at < repoPanels {
			panel.Number = at + 1
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
				panel.Rows = append(panel.Rows, pullRow(u, i, widest, panel.Hidden, th))
			}
		}
		panels = append(panels, panel)
	}
	if len(panels) == 0 && HiddenCount(list) > 0 {
		return note("hidden", th.Muted.Render(tr.EveryOneHidden))
	}
	if len(panels) == 0 {
		return note("none", th.Muted.Render(tr.NoneOpen))
	}
	return panels
}

// HiddenCount is how many repositories of the list the config hides.
func HiddenCount(list *models.Pulls) int {
	n := 0
	if list != nil {
		for _, repo := range list.Repos {
			if len(repo.HiddenBy) > 0 {
				n++
			}
		}
	}
	return n
}

func pullRow(u models.Unit, i, widest int, hidden bool, th theme.Theme) SideRow {
	m := u.Members[i]
	waiting, loaded, bracket := " ", " ", " "
	warning, accent := th.Warning, th.Accent
	if hidden {
		warning, accent = th.Faint, th.Faint
	}
	if len(m.Asked) > 0 {
		waiting = warning.Render("●")
	}
	if m.Review != "" {
		loaded = accent.Render("◆")
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
	number, title, author := th.Muted, th.Text, th.Muted
	if m.Draft || hidden {
		number, title = th.Faint, th.Faint
	}
	if hidden {
		author = th.Faint
	}
	return SideRow{
		Key:   fmt.Sprintf("pull:%d", m.Number),
		Text:  waiting + loaded + " " + bracket + " " + number.Render(fmt.Sprintf("%*s", widest+1, fmt.Sprintf("#%d", m.Number))) + " " + title.Render(m.Title),
		Right: author.Render(ansi.Truncate(m.Author, authorWidth, "…")),
		Item:  Item{Kind: "pull", Pull: m.Number, URL: m.URL, Title: m.Title},
	}
}
