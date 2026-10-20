package presentation

import (
	"fmt"
	"path"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/gui/views"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/theme"
	"github.com/connorb645/lazyreview/internal/utils"
)

// PostLabels are the words for each action posting offers.
func PostLabels(tr *i18n.TranslationSet) map[string]string {
	return map[string]string{"comment": tr.AddComments, "approve-with-comments": tr.ApproveWithComments, "approve": tr.Approve}
}

// PostingBox draws posting's popup for a pull request of a repository: the
// actions offered, the one chosen selected, and, once it is chosen to be
// confirmed, what it sends and the comments it sends.
func PostingBox(tr *i18n.TranslationSet, th theme.Theme, repo string, pull int, m models.QueueMember, actions []string, choice int, confirming bool, width int) []string {
	inner := width - 2
	var out []string
	for i, action := range actions {
		label := PostLabels(tr)[action]
		if action != "approve" {
			label += fmt.Sprintf(" (%d)", len(m.Comments))
		}
		row := " " + th.Key.Render(fmt.Sprintf("%d", i+1)) + "  " + th.Text.Render(label)
		if i == choice {
			row = th.Selected(row, inner, !confirming)
		}
		out = append(out, row)
	}
	footer := OfTotal(tr, choice+1, len(actions))
	if confirming {
		action := actions[choice]
		out = append(out, th.InactiveBorder.Render(strings.Repeat("─", inner)))
		for _, l := range utils.Wrap(m.Sets[action].Words, inner-2) {
			out = append(out, " "+th.Text.Render(l))
		}
		if action != "approve" {
			place := 0
			for _, c := range m.Comments {
				place = max(place, ansi.StringWidth(commentPlace(tr, c)))
			}
			out = append(out, "")
			for i, c := range m.Comments {
				first, _, _ := strings.Cut(c.Body, "\n")
				out = append(out, " "+th.Muted.Render(fmt.Sprintf("%2d  %-*s", i+1, place, commentPlace(tr, c)))+"  "+th.Text.Render(first))
			}
		}
		footer = ""
	}
	title := utils.ResolvePlaceholderString(tr.PostTo, map[string]string{"repo": repo, "pull": utils.Itoa(pull)})
	return views.Framed(out, views.Extent{}, width, len(out)+2, title, "", footer, true, th)
}

func commentPlace(tr *i18n.TranslationSet, c models.Comment) string {
	return path.Base(c.Path) + ", " + LinesWords(tr, c.Side, c.Line, c.StartLine, c.StartSide)
}
