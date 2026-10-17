package presentation

import (
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// SideContent draws a side panel's rows from its first one shown into the
// inside of its frame, the cursor's row selected, the highlight's rows lit
// and those the filter leaves out marked, and says where the cursor is
// among the rows, such as "2 of 5".
func SideContent(tr *i18n.TranslationSet, th theme.Theme, rows []SideRow, cursor, top int, in views.Rect, focused bool) ([]string, string) {
	var content []string
	for i := top; i < top+in.H && i < len(rows); i++ {
		row := rows[i]
		line := sideLine(row, in.W)
		switch {
		case i == cursor && !row.Heading && focused:
			line = th.Selected(line, in.W, true)
		case row.Lit:
			line = th.Visual(line, in.W)
		default:
			if row.Out {
				line = th.Out(line)
			}
			if i == cursor && !row.Heading {
				line = th.Selected(line, in.W, false)
			}
		}
		content = append(content, line)
	}
	at, total := 0, 0
	for i, row := range rows {
		if !row.Heading {
			total++
			if i <= cursor {
				at = total
			}
		}
	}
	return content, OfTotal(tr, at, total)
}

// OfTotal says where one is among others, such as "2 of 5".
func OfTotal(tr *i18n.TranslationSet, at, total int) string {
	return utils.ResolvePlaceholderString(tr.OfTotal, map[string]string{"at": utils.Itoa(at), "total": utils.Itoa(total)})
}

func sideLine(row SideRow, width int) string {
	right := row.Right
	if right != "" {
		right = " " + right + " "
	}
	room := max(1, width-1-ansi.StringWidth(right))
	text := row.Text
	if row.Title != "" {
		text += row.TitleStyle.Render(utils.FitTitle(row.Title, room-ansi.StringWidth(text)))
	}
	return " " + utils.Fit(ansi.Truncate(text, room, "…"), room) + right
}
