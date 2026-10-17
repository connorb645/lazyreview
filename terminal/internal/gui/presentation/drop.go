package presentation

import (
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// DropBox draws the question asked before a queued comment is dropped, in
// a popup width wide, its foot naming the confirmation's key from labels.
func DropBox(tr *i18n.TranslationSet, th theme.Theme, labels KeyLabels, comment, pull, width int) []string {
	body := utils.Wrap(utils.ResolvePlaceholderString(tr.DropQuestion, map[string]string{"comment": utils.Itoa(comment), "pull": utils.Itoa(pull)}), width-4)
	for i, l := range body {
		body[i] = " " + th.Text.Render(l)
	}
	title := utils.ResolvePlaceholderString(tr.DropTitle, map[string]string{"comment": utils.Itoa(comment)})
	foot := utils.ResolvePlaceholderString(tr.DropFooter, map[string]string{"confirm": labels.Of("confirm")})
	return views.Framed(body, views.Extent{}, width, len(body)+2, title, "", foot, true, th)
}
