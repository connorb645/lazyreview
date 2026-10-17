package presentation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)

// HomeMain is the home screen's main panel: its title and its lines, about
// the pull request selected, an account GitHub could not read, or why there
// is nothing to list.
func HomeMain(tr *i18n.TranslationSet, th theme.Theme, md *Markdown, width int, v HomeMainView) (string, []string) {
	b := &builder{width: width, tr: tr, th: th}
	title := tr.PullRequestsTitle
	switch {
	case v.List == nil && v.Failed != "":
		b.text("")
		b.prose(" ", th.Bad, tr.CouldNotReadPulls+": "+v.Failed)
		b.text("")
		b.prose(" ", th.Muted, tr.TryAgainOrQuit)
	case v.List == nil:
		b.text("")
		b.prose(" ", th.Muted, tr.ReadingPulls)
	case len(v.List.Accounts) == 0:
		title = tr.AccountsTitle
		b.text("")
		b.prose(" ", th.Text, utils.ResolvePlaceholderString(tr.NameAccountsIn, map[string]string{"path": fromHome(v.List.Settings, v.Home)}))
		b.text("")
		for _, l := range strings.Split(tr.SettingsSample, "\n") {
			b.text("   " + th.Accent.Render(l))
		}
		b.text("")
		b.prose(" ", th.Muted, tr.EachAccount)
	case v.Chosen:
		title = fmt.Sprintf("%s #%d", v.Repo.Name, v.Pull.Number)
		pullLines(b, md, v)
	case v.Row.Item.Kind == "account":
		title = v.Row.Item.Title
		b.text("")
		b.prose(" ", th.Text, utils.ResolvePlaceholderString(tr.CouldNotReadAs, map[string]string{"login": v.Row.Item.Title}))
		for _, read := range v.List.Accounts {
			if read.Login == v.Row.Item.Title {
				b.prose(" ", th.Bad, read.Error)
			}
		}
		b.text("")
		b.prose(" ", th.Muted, tr.LeftOutUntil)
	default:
		logins := make([]string, len(v.List.Accounts))
		for i, read := range v.List.Accounts {
			logins[i] = read.Login
		}
		b.text("")
		b.prose(" ", th.Text, utils.ResolvePlaceholderString(tr.NothingInvolves, map[string]string{"logins": listed(tr, logins)}))
		b.text("")
		b.prose(" ", th.Muted, tr.ReadsGitHubAgain)
	}
	content := make([]string, len(b.lines))
	for i, l := range b.lines {
		content[i] = l.Text
	}
	return title, content
}

func pullLines(b *builder, md *Markdown, v HomeMainView) {
	th, tr, repo, u, m := b.th, b.tr, v.Repo, v.Unit, v.Pull
	b.prose(" ", th.Bold, m.Title)
	b.parts(" ", th.Muted.Render(utils.ResolvePlaceholderString(tr.PullBy, map[string]string{"pull": utils.Itoa(m.Number), "author": m.Author})), th.Muted.Render(m.Head+" → "+m.Base))
	b.text("")
	if len(m.Asked) > 0 {
		marked(b, th.Warning.Render("●"), th.Text, utils.ResolvePlaceholderString(tr.WaitingOnYou, map[string]string{"who": listed(tr, m.Asked)}))
	}
	if m.Review != "" {
		marked(b, th.Accent.Render("◆"), th.Text, tr.LoadedInLazyreview)
	} else {
		marked(b, " ", th.Muted, tr.NotLoadedYet)
	}
	if m.Yours {
		marked(b, " ", th.Muted, tr.YouOpenedIt)
	}
	if m.Commented {
		marked(b, " ", th.Muted, tr.YouCommented)
	}
	if m.Draft {
		marked(b, " ", th.Muted, tr.ADraft)
	}
	b.text("")
	switch {
	case v.Loading == UnitKey(repo, u):
		b.prose(" ", th.Accent, tr.LoadingFromGitHubNow)
	case u.Review != "" && len(u.Members) > 1:
		b.prose(" ", th.Text, tr.EnterOpensStack)
	case u.Review != "":
		b.prose(" ", th.Text, tr.EnterOpensReview)
	case len(u.Members) > 1:
		pulls := strings.TrimPrefix(UnitName(repo, u), repo.Name+" ")
		b.prose(" ", th.Text, utils.ResolvePlaceholderString(tr.EnterLoadsStack, map[string]string{"pulls": pulls, "account": repo.Account}))
	default:
		b.prose(" ", th.Text, utils.ResolvePlaceholderString(tr.EnterLoadsIt, map[string]string{"account": repo.Account}))
	}
	if len(u.Members) > 1 {
		b.text("", " "+th.Muted.Render(tr.StackBottomFirst))
		for _, other := range u.Members {
			mark, style := "  ", th.Muted
			if other.Number == m.Number {
				mark, style = th.Accent.Render("▸ "), th.Text
			}
			b.text(" " + mark + style.Render(fmt.Sprintf("#%d %s", other.Number, other.Title)))
		}
	}
	b.text("")
	description := strings.TrimSpace(htmlComment.ReplaceAllString(m.Description, ""))
	if description == "" {
		b.text(" " + th.Muted.Render(tr.NoDescription))
		return
	}
	for _, l := range withoutMargin(md.Lines(description, b.width-2)) {
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
