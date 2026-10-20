package presentation

import (
	"github.com/connorb645/lazyreview/internal/gui/views"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/theme"
	"github.com/connorb645/lazyreview/internal/utils"
)

// DeleteBox draws the question asked before a conversation with Claude is
// deleted, naming its lines and its first question, in a popup width wide,
// its foot naming the confirmation's key from labels.
func DeleteBox(tr *i18n.TranslationSet, th theme.Theme, labels KeyLabels, c models.Conversation, width int) []string {
	var body []string
	said := func(words string) {
		for _, l := range utils.Wrap(words, width-4) {
			body = append(body, " "+th.Text.Render(l))
		}
	}
	on := c.Path + ", " + LinesWords(tr, c.Side, c.Line, c.StartLine, c.StartSide)
	said(utils.ResolvePlaceholderString(tr.DeleteQuestion, map[string]string{"on": on}))
	if len(c.Messages) > 0 {
		question := clipped(utils.PlainText(c.Messages[0].Text), width-8, 2)
		question[0] = "“" + question[0]
		question[len(question)-1] += "”"
		for i, l := range question {
			if i > 0 {
				l = " " + l
			}
			body = append(body, "   "+th.Muted.Render(l))
		}
	}
	if Waiting(c) {
		said(tr.DeleteStopsClaude)
	}
	said(tr.DeleteGone)
	foot := utils.ResolvePlaceholderString(tr.DeleteFooter, map[string]string{"confirm": labels.Of("confirm")})
	return views.Framed(body, views.Extent{}, width, len(body)+2, tr.DeleteTitle, "", foot, true, th)
}
